package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Env     string
	DBPath  string
	SSHAddr string
}

func LoadConfig() (*Config, error) {
	err := godotenv.Load("../.env")
	if err != nil {
		return nil, fmt.Errorf("couldn't load .env: %w", err)
	}

	env := os.Getenv("APP_ENV")
	if env != "dev" && env != "prod" {
		return nil, fmt.Errorf("APP_ENV must be \"dev\" or \"prod\", got %q", env)
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		return nil, fmt.Errorf("DB_PATH must be set")
	}

	sshAddr := os.Getenv("SSH_ADDR")
	if env == "prod" && sshAddr == "" {
		return nil, fmt.Errorf("SSH_ADDR must be set when APP_ENV=prod")
	}

	return &Config{
		Env:     env,
		DBPath:  dbPath,
		SSHAddr: sshAddr,
	}, nil
}
