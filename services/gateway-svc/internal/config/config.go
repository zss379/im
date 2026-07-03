package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server      ServerConfig     `yaml:"server"`
	Auth        AuthConfig       `yaml:"auth"`
	RateLimit   RateLimitConfig  `yaml:"rate_limit"`
	CORS        CORSConfig       `yaml:"cors"`
	Routes      []RouteConfig    `yaml:"routes"`
	Prometheus  PromConfig       `yaml:"prometheus"`
	HealthCheck HealthCheckConfig `yaml:"health_check"`
	Log         LogConfig        `yaml:"log"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type AuthConfig struct {
	JWTSecret    string   `yaml:"jwt_secret"`
	PublicPaths  []string `yaml:"public_paths"`
}

type RateLimitConfig struct {
	Enabled    bool  `yaml:"enabled"`
	RefillRate int   `yaml:"refill_rate"`
	BucketSize int   `yaml:"bucket_size"`
}

type CORSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	AllowedMethods []string `yaml:"allowed_methods"`
	AllowedHeaders []string `yaml:"allowed_headers"`
}

type RouteConfig struct {
	Prefix  string `yaml:"prefix"`
	Backend string `yaml:"backend"`
	Timeout string `yaml:"timeout"`
}

type PromConfig struct {
	Enabled bool   `yaml:"enabled"`
	Path    string `yaml:"path"`
}

type HealthCheckConfig struct {
	Interval string `yaml:"interval"`
	Timeout  string `yaml:"timeout"`
}

type LogConfig struct {
	Level string `yaml:"level"`
	JSON  bool   `yaml:"json"`
}

func (r RouteConfig) ParseTimeout() time.Duration {
	if r.Timeout == "" {
		return 30 * time.Second
	}
	d, err := time.ParseDuration(r.Timeout)
	if err != nil {
		return 30 * time.Second
	}
	return d
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
		Server: ServerConfig{Port: 8080},
		Auth: AuthConfig{
			JWTSecret: "dev-secret",
			PublicPaths: []string{
				"POST:/api/v1/auth/login",
				"POST:/api/v1/auth/refresh",
			},
		},
		RateLimit: RateLimitConfig{
			Enabled:    true,
			RefillRate: 50,
			BucketSize: 200,
		},
		CORS: CORSConfig{
			Enabled:        true,
			AllowedOrigins: []string{"*"},
			AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
			AllowedHeaders: []string{"Origin", "Content-Type", "Authorization", "X-Request-ID"},
		},
		Routes: []RouteConfig{
			{Prefix: "/api/v1/auth", Backend: "http://auth-svc:8080", Timeout: "30s"},
			{Prefix: "/api/v1/messages", Backend: "http://message-svc:8081", Timeout: "60s"},
			{Prefix: "/api/v1/bots", Backend: "http://bot-svc:8082", Timeout: "30s"},
			{Prefix: "/api/v1/sessions", Backend: "http://session-svc:8083", Timeout: "30s"},
			{Prefix: "/api/v1/contacts", Backend: "http://contact-svc:8084", Timeout: "30s"},
			{Prefix: "/api/v1/audit", Backend: "http://audit-svc:8085", Timeout: "30s"},
			{Prefix: "/api/v1/groups", Backend: "http://group-svc:8086", Timeout: "30s"},
			{Prefix: "/api/v1/files", Backend: "http://file-svc:8087", Timeout: "120s"},
			{Prefix: "/api/v1/rc", Backend: "http://rc-svc:8088", Timeout: "30s"},
		},
		Prometheus: PromConfig{Enabled: true, Path: "/metrics"},
		HealthCheck: HealthCheckConfig{
			Interval: "30s",
			Timeout:  "5s",
		},
		Log: LogConfig{Level: "debug", JSON: true},
	}
}
