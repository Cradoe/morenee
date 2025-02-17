// The first event after a transfer request has been initiated synchronousely is to debit the sender
// We do this by getting event to this effect.
// Our listeners checks (polling) every 100ms for new event
// We need to make sure the debitting is done with pessimistic lock, to avoid race condition
// A log of this action is submitted in another go routine
// and we then produce a new asynchronous event to credit the recipient
// We retry failed debit 5 times with exponential delays,
// failure after the 5 trial will result in marking the transaction status as "failed"

package worker

import (
	"encoding/json"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/handler"
	"github.com/cradoe/morenee/internal/stream"
)

func (wk *Worker) DebitWorker() {
	consumer, err := wk.kafkaStream.CreateConsumer(&stream.StreamConsumer{
		GroupId: transferDebitGroupID,
		Topic:   TransferDebitTopic,
	})

	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}
	defer consumer.Close()

	maxRetries := 5
	baseRetryDelay := time.Second * 2

	for {
		select {
		case <-wk.ctx.Done():
			log.Println("DebitWorker received cancellation signal, shutting down...")
			return
		default:
			event := consumer.Poll(100)
			switch e := event.(type) {
			case *kafka.Message:
				// We use a goroutine to process each message independently
				// This is to ensure that the worker remains non-blocking and can continue
				// processing new messages.
				// Delay could be caused by network or retry policy,
				// using goroutine here is an option to make sure our worker can
				// attend to other requests while retries are handled in the background.
				go func(msg *kafka.Message) {
					message := msg.Value
					var transferReq handler.InitiatedTransfer
					json.Unmarshal(message, &transferReq)

					retryCount := 0
					for retryCount < maxRetries {
						success := wk.debitAccount(&transferReq)
						if success {
							wk.kafkaStream.ProduceMessage(TransferCreditTopic, string(msg.Value))
							return
						}

						// we can implement retry mechanism when debit fails
						// this will be done with exponential backoff
						// we must have also confirmed that the `wk.debitAccount` function uses database transaction mechanism
						// with a rollback strategy for when something happen.
						retryCount++
						delay := time.Duration(retryCount) * baseRetryDelay
						log.Printf("Debit attempt failed. Retrying in %v... (attempt %d/%d)\n", delay, retryCount, maxRetries)
						time.Sleep(delay)
					}

					// Final failure handling
					log.Printf("Failed to debit account after %d retries. Message: %s\n", maxRetries, message)

					wk.processFailedDebit(&transferReq)
				}(e)

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

func (wk *Worker) debitAccount(transferReq *handler.InitiatedTransfer) bool {
	_, err := wk.db.DebitWallet(transferReq.SenderWalletID, transferReq.Amount)
	if err != nil {
		return false
	}

	// log operation
	go func() {
		_, err = wk.db.CreateActivityLog(&database.ActivityLog{
			UserID:      transferReq.SenderID,
			Entity:      database.ActivityLogTransactionEntity,
			EntityId:    transferReq.ID,
			Description: handler.TransactionActivityLogDebitDescription,
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

func (wk *Worker) processFailedDebit(transferReq *handler.InitiatedTransfer) bool {
	// When debit fails, we would mark the transaction status as failed

	_, err := wk.db.UpdateTransactionStatus(transferReq.ID, database.TransactionStatusFailed)
	if err != nil {
		log.Printf("Error marking transaction as failed: %v", err)
		return false
	}
	// create an activity log to this effect
	_, err = wk.db.CreateActivityLog(&database.ActivityLog{
		UserID:      transferReq.SenderID,
		Entity:      database.ActivityLogTransactionEntity,
		EntityId:    transferReq.ID,
		Description: handler.TransactionActivityLogFailedDebitDescription,
	})

	if err != nil {
		log.Printf("Error logging failed transaction action: %v", err)
	}

	return true
}
