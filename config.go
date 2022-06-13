package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type (
	Config struct {
		State         StateConfig         `yaml:"state"`
		Notifications NotificationsConfig `yaml:"notifications"`
		Commands      []CommandConfig     `yaml:"commands"`
	}

	StateConfig struct {
		Bolt string `yaml:"bolt"`
	}

	NotificationsConfig struct {
		Email *EmailConfig `yaml:"email"`
	}

	EmailConfig struct {
		SMTP     string             `yaml:"smtp"`
		Login    string             `yaml:"login"`
		Password string             `yaml:"password"`
		Groups   []EmailGroupConfig `yaml:"groups"`
	}

	EmailGroupConfig struct {
		Name      string   `yaml:"name"`
		Addresses []string `yaml:"addresses"`
	}

	CommandConfig struct {
		Name           string        `yaml:"name"`
		Exec           string        `yaml:"exec"`
		Threshold      uint          `yaml:"threshold"`
		ThresholdSleep time.Duration `yaml:"threshold_sleep"`
		Cron           string        `yaml:"cron"`
		Interval       time.Duration `yaml:"interval"`
		Timeout        time.Duration `yaml:"timeout"`
		Notifications  []string      `yaml:"notifications"`
	}
)

var Version = "dev"

func ReadConfig(file string) (*Config, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return parseConfig(data)
}

func parseConfig(data []byte) (*Config, error) {
	c := new(Config)
	if err := yaml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return c, validateConfig(c)
}

func validateConfig(cfg *Config) error {
	// Step 1: sanity check of notification groups
	configuredNotifications := make(map[string]map[string]struct{})
	if cfg.Notifications.Email != nil {
		emailMap := make(map[string]struct{}, len(cfg.Notifications.Email.Groups))
		for _, group := range cfg.Notifications.Email.Groups {
			if len(group.Addresses) == 0 {
				return fmt.Errorf("email group %s contains no addresses", group.Name)
			}
			if _, ok := emailMap[group.Name]; ok {
				return fmt.Errorf("non unique email group name %s", group.Name)
			}
			emailMap[group.Name] = struct{}{}
		}
		configuredNotifications["email"] = emailMap
	}
	// With new notification channels, add more sanity checks

	// Step 2: sanity check of command notifications
	for _, cmd := range cfg.Commands {
		for _, n := range cmd.Notifications {
			elements := strings.Split(n, ":")
			if len(elements) != 2 {
				return fmt.Errorf("invalid notification tuple %s in command %s", n, cmd.Name)
			} else if groups, ok := configuredNotifications[elements[0]]; !ok {
				return fmt.Errorf("invalid notification type %s in command %s", elements[0], cmd.Name)
			} else if _, ok = groups[elements[1]]; !ok {
				return fmt.Errorf("invalid notification group %s in command %s", elements[1], cmd.Name)
			}
		}
	}

	return nil
}
