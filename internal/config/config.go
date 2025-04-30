package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	App struct {
		Test     bool `yaml:"test"`
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

func LoadConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	return &cfg, err
}
