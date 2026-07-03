package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	MySQL      MySQLConfig      `yaml:"mysql"`
	Redis      RedisConfig      `yaml:"redis"`
	Kafka      KafkaConfig      `yaml:"kafka"`
	Log         LogConfig         `yaml:"log"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
}

type KafkaConfig struct {
	Brokers             []string `yaml:"brokers"`
	TopicBlockedMessage string   `yaml:"topic_blocked_message"`
	ConsumerGroupID     string   `yaml:"consumer_group_id"`
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
	JSON bool `yaml:"json"`
}

type PrometheusConfig struct {
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
		Server: ServerConfig{
			Port:      8085,
			JWTSecret: "changeme-in-production",
		},
		MySQL: MySQLConfig{
			DSN: "root:123456@tcp(127.0.0.1:3306)/im_audit?charset=utf8mb4&parseTime=True&loc=Local",
		},
		Redis: RedisConfig{
			Addr: "127.0.0.1:6379",
			DB:   5,
		},
		Kafka: KafkaConfig{
			Brokers:             []string{"127.0.0.1:9092"},
			TopicBlockedMessage: "blocked_message",
			ConsumerGroupID:     "audit-svc-blocked-message",
		},
		Prometheus: PrometheusConfig{
			Enabled: true,
			Path:    "/metrics",
		},
	}
}
