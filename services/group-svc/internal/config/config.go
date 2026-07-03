package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig `yaml:"server"`
	MySQL      MySQLConfig  `yaml:"mysql"`
	Redis      RedisConfig  `yaml:"redis"`
	Log        LogConfig    `yaml:"log"`
	Prometheus PromConfig   `yaml:"prometheus"`
}

type ServerConfig struct {
	Port      int    `yaml:"port"`
	JWTSecret string `yaml:"jwt_secret"`
}

type MySQLConfig struct {
	DSN string `yaml:"dsn"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	JSON  bool   `yaml:"json"`
}

type PromConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{Port: 8086, JWTSecret: "dev-secret"},
		MySQL: MySQLConfig{
			DSN: "root:root123@tcp(localhost:3306)/im_shared?charset=utf8mb4&parseTime=True&loc=Local",
		},
		Redis: RedisConfig{Addr: "localhost:6379", Password: "redis123", DB: 0},
		Log:   LogConfig{Level: "debug", JSON: true},
		Prometheus: PromConfig{Enabled: true, Path: "/metrics"},
	}
}
