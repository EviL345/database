package config

import (
	"os"
)

type Config struct {
	Db  Storage
	Srv Server
}

type Server struct {
	Port string
}

type Storage struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
}

func GetConfig() *Config {
	cfg := Config{
		Db: Storage{
			Host:     os.Getenv("DB_HOST"),
			Port:     os.Getenv("DB_PORT"),
			Name:     os.Getenv("DB_NAME"),
			User:     os.Getenv("DB_USER"),
			Password: os.Getenv("DB_PASSWORD"),
		},
		Srv: Server{
			Port: os.Getenv("SERVER_PORT"),
		},
	}

	return &cfg
}
