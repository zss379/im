package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	MySQL    MySQLConfig    `yaml:"mysql"`
	Redis    RedisConfig    `yaml:"redis"`
	Kafka    KafkaConfig    `yaml:"kafka"`
	OpenIM   OpenIMConfig   `yaml:"openim"`
	Webhook  WebhookConfig  `yaml:"webhook"`
	SSE      SSEConfig      `yaml:"sse"`
	Rate     RateConfig     `yaml:"rate"`
	Log      LogConfig      `yaml:"log"`
	Prometheus PromConfig   `yaml:"prometheus"`
}

type ServerConfig struct {
	Port int `yaml:"port" default:"8082"`
}

type MySQLConfig struct {
	DSN string `yaml:"dsn"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type KafkaConfig struct {
	Brokers []string `yaml:"brokers"`
	Topic   string   `yaml:"topic" default:"bot_trigger"`
	GroupID string   `yaml:"group_id" default:"bot-svc"`
}

type OpenIMConfig struct {
	APIEndpoint string `yaml:"api_endpoint"`
	Secret      string `yaml:"secret"`
}

type WebhookConfig struct {
	SyncTimeout   string   `yaml:"sync_timeout" default:"3s"`
	MaxRetries    int      `yaml:"max_retries" default:"3"`
	RetryBackoff  []string `yaml:"retry_backoff" default:"[\"100ms\",\"500ms\",\"1s\"]"`
	MaxBodySizeMB int      `yaml:"max_body_size_mb" default:"1"`
	PendingTTL    string   `yaml:"pending_ttl" default:"30m"`
}

type SSEConfig struct {
	MaxConnections    int    `yaml:"max_connections" default:"1000"`
	IdleTimeout       string `yaml:"idle_timeout" default:"5m"`
	MaxStreamDuration string `yaml:"max_stream_duration" default:"30m"`
}

type RateConfig struct {
	BotMsgPerSec int `yaml:"bot_msg_per_sec" default:"10"`
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
		Server: ServerConfig{Port: 8082},
		MySQL: MySQLConfig{DSN: "root:root123@tcp(localhost:3306)/im_shared?charset=utf8mb4&parseTime=True"},
		Redis: RedisConfig{Addr: "localhost:6379", Password: "redis123", DB: 0},
		Kafka: KafkaConfig{
			Brokers: []string{"localhost:9092"},
			Topic:   "bot_trigger",
			GroupID: "bot-svc",
		},
		OpenIM: OpenIMConfig{
			APIEndpoint: "http://localhost:10002",
			Secret:      "openim-secret",
		},
		Webhook: WebhookConfig{
			SyncTimeout:   "3s",
			MaxRetries:    3,
			RetryBackoff:  []string{"100ms", "500ms", "1s"},
			MaxBodySizeMB: 1,
			PendingTTL:    "30m",
		},
		SSE: SSEConfig{
			MaxConnections:    1000,
			IdleTimeout:       "5m",
			MaxStreamDuration: "30m",
		},
		Rate: RateConfig{BotMsgPerSec: 10},
		Log:  LogConfig{Level: "debug", JSON: true},
		Prometheus: PromConfig{Enabled: true, Path: "/metrics"},
	}
}
