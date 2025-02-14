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
		Topic:   transferCreditTopic,
	})

	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}
	for {
		event := consumer.Poll(100) // Poll every 100ms
		switch e := event.(type) {
		case *kafka.Message:
			message := e.Value
			log.Printf("Credit message received on %s: %s\n", e.TopicPartition, string(e.Value))

			var transferReq handler.InitiatedTransfer
			json.Unmarshal(message, &transferReq)

			success := wk.creditAccount(&transferReq)
			if success {
				log.Printf("Credit completed successfully: %v", transferReq)
				// Produce message the success worker can mark the transaction as successful
				wk.kafkaStream.ProduceMessage(transferSuccessTopic, string(e.Value))
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
			Type:        database.AccountLogTypeTransaction,
			TypeId:      transferReq.ID,
			Description: database.AccountLogTransactionCreditDescription,
		})

		if err != nil {
			log.Printf("Error logging credit action: %v", err)
		}
	}()

	return true
}
