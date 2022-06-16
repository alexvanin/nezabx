package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/alexvanin/nezabx/db"
	"github.com/alexvanin/nezabx/notifications/email"
	"github.com/alexvanin/nezabx/runners"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	configFile := flag.String("c", "", "config file")
	debugFlag := flag.Bool("debug", false, "debug mode")
	versionFlag := flag.Bool("version", false, "show version")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("Nezabx %s\n", Version)
		os.Exit(0)
	}

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	log := Logger(*debugFlag)

	cfg, err := ReadConfig(*configFile)
	if err != nil {
		log.Error("invalid configuration", zap.Error(err))
		os.Exit(1)
	}

	state, err := db.NewBolt(cfg.State.Bolt)
	if err != nil {
		log.Error("invalid configuration", zap.Error(err))
		os.Exit(1)
	}

	var mailNotificator *email.Notificator

	if cfg.Notifications.Email != nil {
		mailNotificator = email.NewNotificator(
			cfg.Notifications.Email.SMTP,
			cfg.Notifications.Email.Login,
			cfg.Notifications.Email.Password,
		)
		for _, group := range cfg.Notifications.Email.Groups {
			mailNotificator.AddGroup(group.Name, group.Addresses)
		}
	}

	for _, command := range cfg.Commands {
		cmd, err := runners.NewCommand(command.Exec)
		if err != nil {
			log.Warn("invalid command configuration", zap.String("command", command.Name), zap.Error(err))
			os.Exit(1)
		}
		var cronSchedule cron.Schedule
		if command.Interval == 0 {
			cronSchedule, err = cron.ParseStandard(command.Cron)
			if err != nil {
				log.Warn("invalid cron configuration", zap.String("command", command.Name), zap.Error(err))
				os.Exit(1)
			}
		}
		commandRunner := runners.CommandRunner{
			Log:             log,
			DB:              state,
			MailNotificator: mailNotificator,
			Command:         cmd,
			Name:            command.Name,
			Threshold:       command.Threshold,
			ThresholdSleep:  command.ThresholdSleep,
			Timeout:         command.Timeout,
			Interval:        command.Interval,
			CronSchedule:    cronSchedule,
			Notifications:   command.Notifications,
		}
		commandRunner.Run(ctx)
	}

	log.Info("application started")

	<-ctx.Done()

	log.Info("application received termination signal")
}

func Logger(debug bool) *zap.Logger {
	logCfg := zap.NewProductionConfig()
	logCfg.Level.SetLevel(zap.InfoLevel)
	logCfg.DisableCaller = true
	logCfg.Encoding = "console"
	logCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	logCfg.DisableStacktrace = true
	if debug {
		logCfg.Level.SetLevel(zap.DebugLevel)
		logCfg.DisableCaller = false
	}
	logger, _ := logCfg.Build()
	return logger
}
