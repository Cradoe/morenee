// Crediting is done when there's a transfer request and debit has been done from the sender's account
// Creditting locks the wallet if the received amount exceeds the wallet limit of the user,
// ... which is controlled by the user's KYC level
// Our listeners checks (polling) every 100ms for new event
// We need to make sure the creditting is done with optimistic lock, to avoid race condition
// A log of this action is submitted in another go routine
// and we then produce a new asynchronous event to mark the transaction as success
package worker

import (
	"encoding/json"
	"log"

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
	for {
		event := consumer.Poll(100)
		switch e := event.(type) {
		case *kafka.Message:
			message := e.Value
			var transferReq handler.InitiatedTransfer
			json.Unmarshal(message, &transferReq)

			success := wk.creditAccount(&transferReq)
			if success {
				// Produce message the success worker can mark the transaction as successful
				wk.kafkaStream.ProduceMessage(TransferSuccessTopic, string(e.Value))
			}
		case kafka.Error:
			log.Printf("Error: %v\n", e)
		default:
			// Handle other events if needed
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
		_, err = wk.db.CreateAccountLog(&database.AccountLog{
			UserID:      transferReq.RecipientID,
			Entity:      database.AccountLogTransactionEntity,
			EntityId:    transferReq.ID,
			Description: database.AccountLogTransactionCreditDescription,
		})

		if err != nil {
			log.Printf("Error logging credit action: %v", err)
			// We should raise a critical error that notifies all concerned parties
			// whenever we encountered failure in logging action.
			// Logging is a key part of our system and should be treated as priority.
		}
	}()

	return true
}
