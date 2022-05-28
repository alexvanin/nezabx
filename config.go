package main

import (
	"fmt"
	"os"
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
	c := new(Config)
	err = yaml.Unmarshal(data, c)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	return c, validateConfig(c)
}

func validateConfig(_ *Config) error {
	// todo(alexvanin): set defaults and validate values such as bad email addresses
	return nil
}
