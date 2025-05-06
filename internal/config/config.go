package config

import (
	"os"
	"time"

	"VK_task/pkg/e"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Env    string `yaml:"env"`
	GRPC   GRPC   `yaml:"grpc"`
	SubPub SubPub `yaml:"sub_pub"`
}

type GRPC struct {
	Addr string `yaml:"addr"`
	Port int    `yaml:"port"`
}

type SubPub struct {
	SubjectBuffer int           `yaml:"subject_buffer"`
	CloseTimeout  time.Duration `yaml:"close_timeout"`
}

func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}

	return cfg
}

func Load(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, e.Wrap("failed to open config file", err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)

	var cfg Config
	err = decoder.Decode(&cfg)
	if err != nil {
		return nil, e.Wrap("failed to parse config file", err)
	}

	return &cfg, nil
}
