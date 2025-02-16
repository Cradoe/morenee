// The first event after a transfer request has been initiated synchronousely is to debit the sender
// We do this by getting event to this effect.
// Our listeners checks (polling) every 100ms for new event
// We need to make sure the debitting is done with optimistic lock, to avoid race condition
// A log of this action is submitted in another go routine
// and we then produce a new asynchronous event to credit the recipient

package worker

import (
	"encoding/json"
	"log"

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
	for {
		event := consumer.Poll(100)
		switch e := event.(type) {
		case *kafka.Message:
			message := e.Value
			log.Printf("Debit message received on %s: %s\n", e.TopicPartition, string(e.Value))

			var transferReq handler.InitiatedTransfer
			json.Unmarshal(message, &transferReq)

			success := wk.debitAccount(&transferReq)
			if success {
				log.Printf("Debit completed successfully: %v", transferReq)
				// Produce message so that the credit worker can credit the receiver
				wk.kafkaStream.ProduceMessage(TransferCreditTopic, string(e.Value))
			}
		case kafka.Error:
			log.Printf("Error: %v\n", e)
		default:
			// Handle other events if needed
		}
	}

}

func (wk *Worker) debitAccount(transferReq *handler.InitiatedTransfer) bool {
	_, err := wk.db.DebitWallet(transferReq.SenderWalletID, transferReq.Amount)
	if err != nil {
		log.Printf("Error debitting wallet: %v", err)
		return false
	}

	// log operation
	go func() {
		_, err = wk.db.CreateAccountLog(&database.AccountLog{
			UserID:      transferReq.SenderID,
			Entity:      database.AccountLogTransactionEntity,
			EntityId:    transferReq.ID,
			Description: database.AccountLogTransactionDebitDescription,
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
