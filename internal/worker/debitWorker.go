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
		Topic:   transferDebitTopic, // Listen to debit the sender's account
	})

	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}
	for {
		event := consumer.Poll(100) // Poll every 100ms
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
				wk.kafkaStream.ProduceMessage(transferCreditTopic, string(e.Value))
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
			Type:        database.AccountLogTypeTransaction,
			TypeId:      transferReq.ID,
			Description: database.AccountLogTransactionDebitDescription,
		})

		if err != nil {
			log.Printf("Error logging debit action: %v", err)
		}
	}()

	return true
}
