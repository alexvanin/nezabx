package runners

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/alexvanin/nezabx/db"
	"github.com/alexvanin/nezabx/notifications/email"
	"github.com/google/shlex"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

type (
	Command struct {
		orig string
		cmd  string
		args []string
	}

	CommandRunner struct {
		Log             *zap.Logger
		DB              *db.Bolt
		MailNotificator *email.Notificator
		Command         *Command
		Name            string
		Threshold       uint
		ThresholdSleep  time.Duration
		Timeout         time.Duration
		Interval        time.Duration
		CronSchedule    cron.Schedule
		Notifications   []string
	}
)

func NewCommand(path string) (*Command, error) {
	command, err := shlex.Split(path)
	if err != nil {
		return nil, err
	}
	if len(command) < 1 {
		return nil, errors.New("empty command")
	}
	return &Command{
		orig: path,
		cmd:  command[0],
		args: command[1:],
	}, nil
}

func (h Command) Exec() ([]byte, error) {
	cmd := exec.Command(h.cmd, h.args...)
	return cmd.CombinedOutput()
}

func (h Command) String() string {
	return h.orig
}

func (c CommandRunner) Run(ctx context.Context) {
	h := sha256.Sum256([]byte(c.Name))
	id := h[:]

	go func() {
		firstIter := true
		for {
			if !firstIter || c.CronSchedule == nil {
				c.run(ctx, id)
			}
			firstIter = false
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.untilNextIteration()):
			}
		}
	}()
}

func (c CommandRunner) run(ctx context.Context, id []byte) {
	st, err := c.DB.Status(id)
	if err != nil {
		c.Log.Warn("database is broken", zap.String("id", hex.EncodeToString(id)), zap.Error(err))
		return
	}

	c.Log.Debug("starting script", zap.Stringer("cmd", c.Command))
	output, err := execScript(ctx, c.Command, c.Timeout)
	if err == nil {
		c.processSuccessfulExecution(id, st)
		return
	}

	if !st.Notified {
		for i := 1; i < int(c.Threshold); i++ {
			c.Log.Info("script run failed", zap.Stringer("cmd", c.Command), zap.Int("iteration", i))
			select {
			case <-ctx.Done():
				return
			case <-time.After(c.ThresholdSleep):
			}

			output, err = execScript(ctx, c.Command, c.Timeout)
			if err == nil {
				c.processSuccessfulExecution(id, st)
				return
			}
		}
	}

	c.processFailedExecution(id, st, output, err)
}

func (c CommandRunner) processSuccessfulExecution(id []byte, st db.Status) {
	if !st.Failed {
		c.Log.Info("script run ok", zap.Stringer("cmd", c.Command),
			zap.Time("next iteration at", c.nextIteration()))
		return
	}

	c.Log.Info("script run ok, recovered after failure", zap.Stringer("cmd", c.Command),
		zap.Time("next iteration at", c.nextIteration()))

	st.Failed = false
	st.Notified = false
	err := c.DB.SetStatus(id, st)
	if err != nil {
		c.Log.Warn("database is broken", zap.String("id", hex.EncodeToString(id)), zap.Error(err))
	}
}

func (c CommandRunner) processFailedExecution(id []byte, st db.Status, output []byte, err error) {
	if st.Failed {
		c.Log.Info("script run failed, notification has already been sent",
			zap.Stringer("cmd", c.Command),
			zap.Time("next iteration at", c.nextIteration()))
		return
	}

	c.Log.Info("script run failed, sending notification",
		zap.Stringer("cmd", c.Command),
		zap.Time("next iteration at", c.nextIteration()))
	st.Failed = true
	st.Notified = true
	err = c.notify(output, err)
	if err != nil {
		c.Log.Warn("notification was not sent", zap.Stringer("cmd", c.Command), zap.Error(err))
		st.Notified = false
	}

	err = c.DB.SetStatus(id, st)
	if err != nil {
		c.Log.Warn("database is broken", zap.String("id", hex.EncodeToString(id)), zap.Error(err))
	}
}

func (c CommandRunner) notify(out []byte, err error) error {
	msg := fmt.Sprintf("Script runner <b>\"%s\"</b> has failed.<br>"+
		"Executed command: <b>%s</b><br>"+
		"Exit error: <b>%s</b><br><br>"+
		"Terminal output:<pre>%s</pre>", c.Name, c.Command, err.Error(), out)

	for _, target := range c.Notifications {
		kv := strings.Split(target, ":")
		if len(kv) != 2 {
			c.Log.Warn("invalid notification target", zap.String("value", target))
			continue
		}
		switch kv[0] {
		case "email":
			if c.MailNotificator == nil {
				c.Log.Warn("email notifications were not configured")
				continue
			}
			err = c.MailNotificator.Send(kv[1], "Nezabx alert message", msg)
			if err != nil {
				return err
			}
		default:
			c.Log.Warn("invalid notification type", zap.String("value", target))
			continue
		}
	}
	return nil
}

func (c CommandRunner) untilNextIteration() time.Duration {
	if c.CronSchedule != nil {
		return time.Until(c.CronSchedule.Next(time.Now()))
	}
	return c.Interval
}

func (c CommandRunner) nextIteration() time.Time {
	if c.CronSchedule != nil {
		return c.CronSchedule.Next(time.Now()) // no truncate because cron operates at most with minute truncated values
	}
	return time.Now().Add(c.Interval).Truncate(time.Second)
}

func execScript(ctx context.Context, cmd *Command, timeout time.Duration) (output []byte, err error) {
	type cmdOutput struct {
		out []byte
		err error
	}

	res := make(chan cmdOutput)
	go func(ch chan<- cmdOutput) {
		v, err := cmd.Exec()
		res <- cmdOutput{
			out: v,
			err: err,
		}
		close(res)
	}(res)

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, errors.New("script execution timeout")
	case v := <-res:
		return v.out, v.err
	}
}
