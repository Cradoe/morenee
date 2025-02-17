// Crediting is done when there's a transfer request and debit has been done from the sender's account
// Creditting locks the wallet if the received amount exceeds the wallet limit of the user,
// ... which is controlled by the user's KYC level
// Our listeners checks (polling) every 100ms for new event
// We need to make sure the creditting is done with pessimistic lock, to avoid race condition
// A log of this action is submitted in another go routine
// and we then produce a new asynchronous event to mark the transaction as success
package worker

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/handler"
	"github.com/cradoe/morenee/internal/stream"
)

func (wk *Worker) CreditWorker() {
	consumer, err := wk.kafkaStream.CreateConsumer(&stream.StreamConsumer{
		GroupId: transferCreditGroupID,
		Topic:   TransferCreditTopic,
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
			log.Println("CreditWorker received cancellation signal, shutting down...")
			return
		default:
			// Poll for events
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
						success := wk.creditAccount(&transferReq)
						if success {
							// Produce message so the success worker can mark the transaction as successful
							wk.kafkaStream.ProduceMessage(TransferSuccessTopic, string(e.Value))
							return
						}

						// we can implement retry mechanism when credit fails
						// this will be done with exponential backoff
						// we must have also confirmed that the `db.CreditWallet` function uses database transaction mechanism
						// with a rollback strategy for when something happen.
						retryCount++
						delay := time.Duration(retryCount) * baseRetryDelay
						log.Printf("Credit attempt failed. Retrying in %v... (attempt %d/%d)\n", delay, retryCount, maxRetries)
						time.Sleep(delay)
					}

					// Final failure handling
					log.Printf("Failed to credit account after %d retries. Message: %s\n", maxRetries, message)

					wk.processFailedCredit(&transferReq)
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

func (wk *Worker) creditAccount(transferReq *handler.InitiatedTransfer) bool {
	_, err := wk.db.CreditWallet(transferReq.RecipientWalletID, transferReq.Amount)
	if err != nil {
		log.Printf("Error crediting wallet: %v", err)
		return false
	}

	// log operation
	go func() {
		_, err = wk.db.CreateActivityLog(&database.ActivityLog{
			UserID:      transferReq.RecipientID,
			Entity:      database.ActivityLogTransactionEntity,
			EntityId:    transferReq.ID,
			Description: handler.TransactionActivityLogCreditDescription,
		})

		if err != nil {
			log.Printf("Error logging credit action: %v", err)
			// We should raise a critical error that notifies all concerned parties
			// whenever we encountered failure in logging action.
			// Logging is a key part of our system and should be treated as priority.
		}
	}()

	// check and lock account if balance has exceeded balance limit
	go func() {
		wallet, found, err := wk.db.GetWallet(transferReq.RecipientWalletID)
		if err != nil {
			log.Printf("Error getting account limit: %v", err)
		}

		if !found {
			return
		}

		if wallet.Balance > wallet.MaxBalance {
			err = wk.db.LockWallet(transferReq.RecipientWalletID)
			if err != nil {
				log.Printf("Error holding recipient account over max balance limit: %v", err)
			}
		}
	}()

	return true
}

// processFailedCredit handles the reversal of a failed credit transaction.
//
// When a credit transaction fails after multiple retry attempts, this function performs the following steps:
// 1. Logs the failed credit attempt to create a record of the failure.
// 2. Credits the money back to the sender’s wallet to ensure no loss of funds.
// 3. Logs the successful reversal for transparency and tracking purposes.
// 4. Marks the original transaction as "Reversed" to indicate its failure and refund status.
// 5. Creates a new transaction record for the reversal to ensure proper audit trails.
// 6. Logs the new reversal transaction to document the entire process.
func (wk *Worker) processFailedCredit(transferReq *handler.InitiatedTransfer) bool {
	// Log the failed credit attempt synchronously
	_, err := wk.db.CreateActivityLog(&database.ActivityLog{
		UserID:      transferReq.SenderID,
		Entity:      database.ActivityLogTransactionEntity,
		EntityId:    transferReq.ID,
		Description: handler.TransactionActivityLogFailedCreditDescription,
	})
	if err != nil {
		log.Printf("Error logging failed credit action: %v", err)
	}

	// Reverse the money to the sender
	_, err = wk.db.CreditWallet(transferReq.SenderWalletID, transferReq.Amount)
	if err != nil {
		log.Printf("Error reversing money from failed credit: %v", err)
		return false
	}

	// Log the successful credit reversal
	_, err = wk.db.CreateActivityLog(&database.ActivityLog{
		UserID:      transferReq.SenderID,
		Entity:      database.ActivityLogTransactionEntity,
		EntityId:    transferReq.ID,
		Description: handler.TransactionActivityLogCreditDescription,
	})
	if err != nil {
		log.Printf("Error logging credit reversal action: %v", err)
	}

	// Mark the original transaction as reversed
	_, err = wk.db.UpdateTransactionStatus(transferReq.ID, database.TransactionStatusReversed)
	if err != nil {
		log.Printf("Error marking transaction as reversed: %v", err)
		return false
	}

	// Create a new transaction for the reversal
	desc := fmt.Sprintf("Reversal of %f", transferReq.Amount)
	newTrans := &database.Transaction{
		SenderWalletID:    transferReq.SenderWalletID,
		RecipientWalletID: transferReq.SenderWalletID, // sender is the recipient in a reversal
		Amount:            transferReq.Amount,
		ReferenceNumber:   transferReq.ReferenceNumber,
		Description:       sql.NullString{String: desc, Valid: true},
	}
	transaction, err := wk.db.CreateTransaction(newTrans, nil)
	if err != nil {
		log.Printf("Error creating reversal transaction: %v", err)
		return false
	}

	// Log the reversal transaction
	_, err = wk.db.CreateActivityLog(&database.ActivityLog{
		UserID:      transferReq.SenderID,
		Entity:      database.ActivityLogTransactionEntity,
		EntityId:    transaction.ID,
		Description: handler.TransactionActivityLogRevertedDescription,
	})
	if err != nil {
		log.Printf("Error logging reversal transaction action: %v", err)
	}

	return true
}
