package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	MongoDB  MongoDBConfig  `yaml:"mongodb"`
	Redis    RedisConfig    `yaml:"redis"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	OpenIM   OpenIMConfig   `yaml:"openim"`
	Rate     RateConfig     `yaml:"rate"`
	RC       RcConfig       `yaml:"rc"`
	Log      LogConfig      `yaml:"log"`
	Prometheus PromConfig   `yaml:"prometheus"`
}

type RcConfig struct {
	Addr    string        `yaml:"addr"`
	Timeout time.Duration `yaml:"timeout"`
}

type ServerConfig struct {
	Port int `yaml:"port" default:"8081"`
}

type MongoDBConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type KafkaConfig struct {
	Brokers            []string `yaml:"brokers"`
	TopicMessagePush   string   `yaml:"topic_message_push" default:"message_push"`
	TopicBotTrigger    string   `yaml:"topic_bot_trigger" default:"bot_trigger"`
	TopicBlockedMessage string  `yaml:"topic_blocked_message" default:"blocked_message"`
}

type OpenIMConfig struct {
	APIEndpoint string `yaml:"api_endpoint"`
	Secret      string `yaml:"secret"`
}

type RateConfig struct {
	UserMsgPerSec int `yaml:"user_msg_per_sec" default:"5"`
	BotMsgPerSec  int `yaml:"bot_msg_per_sec" default:"10"`
}

type LogConfig struct {
	Level string `yaml:"level" default:"debug"`
	JSON  bool   `yaml:"json" default:"true"`
}

type PromConfig struct {
	Enabled bool   `yaml:"enabled" default:"true"`
	Path    string `yaml:"path" default:"/metrics"`
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
		Server: ServerConfig{Port: 8081},
		MongoDB: MongoDBConfig{
			URI:      "mongodb://root:root123@localhost:27017/im_prod?authSource=admin",
			Database: "im_prod",
		},
		Redis: RedisConfig{Addr: "localhost:6379", Password: "redis123", DB: 0},
		Kafka: KafkaConfig{
			Brokers:          []string{"localhost:9092"},
			TopicMessagePush: "message_push",
			TopicBotTrigger:  "bot_trigger",
			TopicBlockedMessage: "blocked_message",
		},
		RC: RcConfig{
			Addr:    "rc-svc:8088",
			Timeout: 2 * time.Second,
		},
		OpenIM: OpenIMConfig{
			APIEndpoint: "http://localhost:10002",
			Secret:      "openim-secret",
		},
		Rate: RateConfig{UserMsgPerSec: 5, BotMsgPerSec: 10},
		Log:  LogConfig{Level: "debug", JSON: true},
		Prometheus: PromConfig{Enabled: true, Path: "/metrics"},
	}
}
