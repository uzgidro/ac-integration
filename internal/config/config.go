package config

import (
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env          string `yaml:"env" required:"true"`
	HttpServer   `yaml:"http_server"`
	MySQLConnect `yaml:"mysql" required:"true"`
}

type HttpServer struct {
	AllowedOrigins []string      `yaml:"allowed_origins"`
	Address        string        `yaml:"address" env-default:":9010"`
	Timeout        time.Duration `yaml:"timeout" env-default:"4s"`
	IdleTimeout    time.Duration `yaml:"idle-timeout" env-default:"60s"`
}

type MySQLConnect struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	SSLMode  string `yaml:"sslmode"`
}

func MustLoad() *Config {
	configPath := "./config/config.yaml"

	// check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("CONFIG_PATH does not exist: %s", configPath)
	}

	var config Config

	if err := cleanenv.ReadConfig(configPath, &config); err != nil {
		log.Fatalf("Cannot read config: %s", err)
	}

	return &config
}
