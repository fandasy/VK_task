package main

import (
	"VK_task/internal/app"
	"VK_task/internal/config"
	"VK_task/internal/pkg/logger"
	"VK_task/internal/pkg/logger/sl"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	time.Sleep(1 * time.Minute)

	cfg := config.MustLoad(mustGetConfigPath())

	log := logger.Setup(cfg.Env)

	log.Debug("Config", slog.Any("data", cfg))

	// App
	application := app.New(log, cfg.GRPC.Addr, cfg.GRPC.Port, cfg.SubPub.SubjectBuffer)

	go application.MustRun()

	// Waiting for the stop sign
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	sign := <-stop
	log.Info("Application stopping", slog.Any("signal", sign))

	// Graceful Stop
	err := application.StopWithLog(cfg.SubPub.CloseTimeout, log)
	if err != nil {
		log.Error("Failed to stop application", sl.Err(err))
	}

	log.Info("App shutdown")
}

func mustGetConfigPath() string {
	var path string

	flag.StringVar(&path,
		"config",
		"",
		"config file path",
	)
	flag.Parse()

	if path == "" {
		path = os.Getenv("CONFIG_PATH")
		if path == "" {
			panic("config file path is empty")
		}
	}

	return path
}
