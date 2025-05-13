package config

import (
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Token    string `yaml:"token"`
		Test     bool   `yaml:"test"`
		Database struct {
			Type  string `yaml:"type"`
			MySQL struct {
				Host     string `yaml:"host"`
				Port     int    `yaml:"port"`
				User     string `yaml:"user"`
				Password string `yaml:"password"`
				DBName   string `yaml:"dbname"`
			} `yaml:"mysql"`
			SQLite struct {
				Path string `yaml:"path"`
			} `yaml:"sqlite"`
			Postgres struct {
				Host     string `yaml:"host"`
				Port     int    `yaml:"port"`
				User     string `yaml:"user"`
				Password string `yaml:"password"`
				DBName   string `yaml:"dbname"`
				SSLMode  string `yaml:"sslmode"`
			} `yaml:"postgres"`
		} `yaml:"database"`
		Stickers struct {
			Positive []string `yaml:"positive"`
			Negative []string `yaml:"negative"`
			Transfer []string `yaml:"transfer"`
		} `yaml:"stickers"`
		Capitalist struct {
			InitialBalance int `yaml:"initial_balance"`
		} `yaml:"capitalist"`
	} `yaml:"app"`
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
