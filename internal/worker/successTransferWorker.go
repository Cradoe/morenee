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
	defer consumer.Close() // Ensure cleanup

	for {
		select {
		case <-wk.ctx.Done():
			log.Println("SuccessTransferWorker received cancellation signal, shutting down...")
			return
		default:
			// Poll for Kafka events
			event := consumer.Poll(100)
			switch e := event.(type) {
			case *kafka.Message:
				message := e.Value
				var transferReq handler.InitiatedTransfer
				json.Unmarshal(message, &transferReq)

				success := wk.completeTransferOperation(&transferReq)
				if success {
					// Send notifications to the sender and receiver
					log.Printf("Transfer completed successfully: %v", transferReq)
				}
			case kafka.Error:
				log.Printf("Error: %v\n", e)
			case *kafka.AssignedPartitions:
				consumer.Assign(e.Partitions)
			case *kafka.RevokedPartitions:
				consumer.Unassign()
			}
		}
	}
}

func (wk *Worker) completeTransferOperation(transferReq *handler.InitiatedTransfer) bool {
	_, err := wk.db.UpdateTransactionStatus(transferReq.ID, database.TransactionStatusCompleted)
	if err != nil {
		log.Printf("Error updating transaction status: %v", err)
		return false
	}

	_, err = wk.db.CreateActivityLog(&database.ActivityLog{
		UserID:      transferReq.SenderID,
		Entity:      database.ActivityLogTransactionEntity,
		EntityId:    transferReq.ID,
		Description: handler.TransactionActivityLogSuccessDescription,
	})

	if err != nil {
		log.Printf("Error logging debit action: %v", err)
	}

	return true
}
