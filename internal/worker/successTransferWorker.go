// Successful transactions are the ones that as gone through debitting (sender) and creditting (recipient)
// A record was created in the transactions table synchronousely when the transfer was initiated
// We need to mark that record as successful.
// We also need to send necessary notifications to both involed users.
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
		Topic:   TransferSuccessTopic,
	})

	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}
	for {
		event := consumer.Poll(100) // Poll every 100ms
		switch e := event.(type) {
		case *kafka.Message:
			message := e.Value
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

	go func() {
		_, err = wk.db.CreateAccountLog(&database.AccountLog{
			UserID:      transferReq.SenderID,
			Entity:      database.AccountLogTransactionEntity,
			EntityId:    transferReq.ID,
			Description: database.AccountLogTransactionSuccessDescription,
		})

		if err != nil {
			log.Printf("Error logging debit action: %v", err)
			// We should raise a critical error that notifies all concerned parties
			// whenever we encountered failure in logging action.
			// Logging is a key part of our system and should be treated as priority.
		}
	}()

	return true
}
