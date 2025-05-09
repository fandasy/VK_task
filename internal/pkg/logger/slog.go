package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func MustSetup(env, filename string) *slog.Logger {
	log, err := Setup(env, filename)
	if err != nil {
		panic(err)
	}

	return log
}

func Setup(env, filename string) (*slog.Logger, error) {
	var (
		log *slog.Logger
		w   io.Writer
		err error
	)

	if filename != "" {
		w, err = createLogFile(filename)
		if err != nil {
			return nil, err
		}
	} else {
		w = os.Stdout
	}

	switch env {
	case envLocal:
		log = slog.New(
			slog.NewTextHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(w, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.Default()
	}

	return log, nil
}

func createLogFile(filename string) (*os.File, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if err := os.Mkdir(filename, 0774); err != nil {
			return nil, fmt.Errorf("can't create a logs dir: %w", err)
		}
	}

	nowDate := time.Now().Format(time.DateOnly)
	nowTime := strings.ReplaceAll(time.Now().Format(time.TimeOnly), ":", ".")

	file, err := os.Create(filename + "/" + nowDate + "_" + nowTime + ".log")
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return file, nil
}
