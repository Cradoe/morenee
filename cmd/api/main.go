package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/cradoe/morenee/internal/app"
	"github.com/cradoe/morenee/internal/version"
	"github.com/cradoe/morenee/internal/worker"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	err := run(logger)
	if err != nil {
		trace := string(debug.Stack())
		logger.Error(err.Error(), "trace", trace)
		os.Exit(1)
	}

	select {}
}

func run(logger *slog.Logger) error {
	showVersion := flag.Bool("version", false, "display version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("version: %s\n", version.Get())
		return nil
	}

	application, err := app.NewApplication(logger)
	if err != nil {
		return err
	}
	defer application.DB.Close()

	go worker.DebitWorker(application.Kafka)

	return application.ServeHTTP()
}
