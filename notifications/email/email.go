package email

import (
	"errors"
	"fmt"
	"net/smtp"
	"strings"
)

type (
	Sender struct {
		SMTPServer string
		Login      string
		Password   string
	}

	Notificator struct {
		Sender Sender
		Groups map[string][]string
	}
)

func NewNotificator(smtp, login, password string) *Notificator {
	return &Notificator{
		Sender: Sender{
			SMTPServer: smtp,
			Login:      login,
			Password:   password,
		},
		Groups: make(map[string][]string),
	}
}

func (n *Notificator) AddGroup(name string, addresses []string) {
	n.Groups[name] = addresses
}

func (n Notificator) Send(group, sub, body string) error {
	receivers, ok := n.Groups[group]
	if !ok {
		return errors.New("unknown group")
	}
	return n.Sender.SendMail(receivers, sub, body)
}

func (s Sender) SendMail(to []string, subj, body string) error {
	toTxt := fmt.Sprintf("To: %s\r\n", strings.Join(to, ";"))
	subjTxt := fmt.Sprintf("Subject: %s\r\n", subj)
	mimeTxt := "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"
	bodyTxt := fmt.Sprintf("%s\r\n", body)
	mailTxt := toTxt + subjTxt + mimeTxt + bodyTxt
	return smtp.SendMail(s.SMTPServer, s, s.Login, to, []byte(mailTxt))
}

func (s Sender) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (s Sender) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch str := string(fromServer); str {
		case "Username:":
			return []byte(s.Login), nil
		case "Password:":
			return []byte(s.Password), nil
		default:
			return nil, fmt.Errorf("unknown fromServer %s", str)
		}
	}
	return nil, nil
}
