package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server     ServerConfig     `yaml:"server"`
	MySQL      MySQLConfig      `yaml:"mysql"`
	Redis      RedisConfig      `yaml:"redis"`
	MinIO      MinIOConfig      `yaml:"minio"`
	Log         LogConfig         `yaml:"log"`
	Prometheus PrometheusConfig `yaml:"prometheus"`
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

type MinIOConfig struct {
	Endpoint  string `yaml:"endpoint"`
	AccessKey string `yaml:"access_key"`
	SecretKey string `yaml:"secret_key"`
	Bucket    string `yaml:"bucket"`
	UseSSL    bool   `yaml:"use_ssl"`
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
			Port:      8087,
			JWTSecret: "changeme-in-production",
		},
		MySQL: MySQLConfig{
			DSN: "root:123456@tcp(127.0.0.1:3306)/im_file?charset=utf8mb4&parseTime=True&loc=Local",
		},
		Redis: RedisConfig{
			Addr: "127.0.0.1:6379",
			DB:   6,
		},
		MinIO: MinIOConfig{
			Endpoint:  "127.0.0.1:9000",
			AccessKey: "minioadmin",
			SecretKey: "minioadmin",
			Bucket:    "im-files",
		},
		Prometheus: PrometheusConfig{
			Enabled: true,
			Path:    "/metrics",
		},
	}
}
