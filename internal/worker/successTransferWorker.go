package worker

import (
	"encoding/json"
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/handler"
	"github.com/cradoe/morenee/internal/stream"
)

func (wk *Worker) SuccessTransferWorker() {
	consumer, err := wk.kafkaStream.CreateConsumer(&stream.StreamConsumer{
		GroupId: transferSuccessGroupID,
		Topic:   transferCreatedTopic, // Listen to when recipient account has been credited
	})

	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}
	for {
		event := consumer.Poll(100) // Poll every 100ms
		switch e := event.(type) {
		case *kafka.Message:
			message := e.Value
			log.Printf("Success message received on %s: %s\n", e.TopicPartition, string(e.Value))

			var transferReq handler.InitiatedTransfer
			json.Unmarshal(message, &transferReq)

			success := wk.completeTransferOperation(&transferReq)
			if success {
				// send notifications to the sender and receiver
				log.Printf("Transfer completed successfully: %v", transferReq)
			}
		case kafka.Error:
			log.Printf("Error: %v\n", e)
		default:
			// Handle other events if needed
		}
	}

}

func (wk *Worker) completeTransferOperation(transferReq *handler.InitiatedTransfer) bool {
	_, err := wk.db.UpdateTransactionStatus(transferReq.ID, database.TransactionStatusCompleted)
	if err != nil {
		log.Printf("Error updating transaction status: %v", err)
		return false
	}

	// log operation
	_, err = wk.db.CreateTransactionLog(
		&database.TransactionLog{
			UserID:        transferReq.RecipientWalletID,
			TransactionID: transferReq.ID,
			Action:        database.TransactionLogActionSuccess,
		},
	)

	if err != nil {
		log.Printf("Error creating transaction log: %v", err)
		return false
	}

	return true
}
