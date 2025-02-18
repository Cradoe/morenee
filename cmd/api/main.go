package main

import (
	"context"
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
	// Let's ensure the database connection is properly closed when the application ends
	defer application.DB.Close()

	// These topics are required to ensure that messages for various events (e.g., transfer debit, credit, success)
	// are properly published and consumed without errors.

	workerTopics := []string{worker.TransferDebitTopic, worker.TransferCreditTopic, worker.TransferSuccessTopic}
	// Ensure that the specified Kafka topics exist before producing or consuming messages.
	// This step is important to avoid runtime errors or message loss due to missing topics.
	err = application.Kafka.EnsureTopicsExist(workerTopics)
	if err != nil {
		return err
	}

	// Create a cancellable context for managing the application lifecycle
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start workers

	wk := worker.New(&worker.Worker{
		KafkaStream: application.Kafka,
		DB:          application.DB,
		Ctx:         ctx,
		Helper:      application.Helper,
	})

	// In order to simplify things and reduce latency for user during transfer
	// we have set up workers to handle every bits of the process
	// The `HandleTransferMoney` handler function initiates the transaction and produces an event
	// ... that would be received by our first worker `DebitWorker`.
	// We have chosen to have different workers for each of the prcesses rather than having
	// ... all the work done in a single worker.
	// This approach makes it easy to keep each worker simple and focus on doing o
	go wk.DebitWorker()
	go wk.CreditWorker()
	go wk.SuccessTransferWorker()

	err = application.ServeHTTP()
	if err != nil {
		logger.Error("HTTP server error", "error", err)
	}

	return nil
}
