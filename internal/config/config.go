package config

import (
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App AppConfig `yaml:"app"`
}

type AppConfig struct {
	Token         string              `yaml:"token"`
	Test          bool                `yaml:"test"`
	Database      DatabaseConfig      `yaml:"database"`
	Stickers      StickersConfig      `yaml:"stickers"`
	Capitalist    CapitalistConfig    `yaml:"capitalist"`
	ActivityCheck ActivityCheckConfig `yaml:"activity_check"`
}

type DatabaseConfig struct {
	Type     string         `yaml:"type"`
	MySQL    MySQLConfig    `yaml:"mysql"`
	SQLite   SQLiteConfig   `yaml:"sqlite"`
	Postgres PostgresConfig `yaml:"postgres"`
}

type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type PostgresConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

type StickersConfig struct {
	Positive []string `yaml:"positive"`
	Negative []string `yaml:"negative"`
	Transfer []string `yaml:"transfer"`
}

type CapitalistConfig struct {
	InitialBalance int `yaml:"initial_balance"`
}

type ActivityCheckConfig struct {
	Schedule        string         `yaml:"schedule"`
	ResponseTimeout int            `yaml:"response_timeout"`
	MaxRetries      int            `yaml:"max_retries"`
	RetryInterval   int            `yaml:"retry_interval"`
	Channels        ChannelsConfig `yaml:"channels"`
	Rewards         RewardsConfig  `yaml:"rewards"`
}

type ChannelsConfig struct {
	Alerts   string `yaml:"alerts"`
	Warnings string `yaml:"warnings"`
}

type RewardsConfig struct {
	AliveScore int `yaml:"alive_score"`
}

func substituteEnvVars(cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(`\$\{([A-Za-z0-9_]+)\}`)

	dataStr := re.ReplaceAllStringFunc(string(data), func(s string) string {
		match := re.FindStringSubmatch(s)
		if len(match) == 2 {
			if val := os.Getenv(match[1]); val != "" {
				return val
			}
		}
		return s
	})
	return yaml.Unmarshal([]byte(dataStr), cfg)
}

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, err
	}
	if err := substituteEnvVars(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
