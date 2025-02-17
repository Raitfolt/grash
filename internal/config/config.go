package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"go.uber.org/zap"
)

type Config struct {
	Env             string        `yaml:"env" env-default:"local"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout" env-default:""`
	HTTPServer      `yaml:"http_server"`
}

type HTTPServer struct {
	Address string `yaml:"address" env-default:"localhost:8080"`
}

func MustLoad(log *zap.Logger) *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}
	logAbsPath, err := filepath.Abs(configPath)
	if err != nil {
		log.Fatal("failed to resolve absolute path", zap.String("path", configPath), zap.Error(err))
	}
	log.Info("absolute config path", zap.String("path", logAbsPath))
	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatal("config file does not exist", zap.String("path", configPath))
	}

	var cfg Config

	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatal("cannot read config", zap.String("error", err.Error()))
	}

	return &cfg
}
