package main

import (
	"log/slog"
	"os"
)

func initSlog() (*os.File, error) {
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "DEBUG" {
		logLevel = slog.LevelDebug
	}

	file, err := os.OpenFile("debug.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{Level: logLevel})))

	return file, nil
}
