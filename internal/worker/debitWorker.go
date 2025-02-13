package worker

import (
	"encoding/json"
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/cradoe/morenee/internal/handler"
	"github.com/cradoe/morenee/internal/stream"
)

func DebitWorker(kafkaStream *stream.KafkaStream) {
	consumer, err := kafkaStream.CreateConsumer(&stream.StreamConsumer{
		GroupId: "debit-group",
		Topic:   "transfer.created",
	})

	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}
	for {
		event := consumer.Poll(100) // Poll every 100ms
		switch e := event.(type) {
		case *kafka.Message:
			message := e.Value
			log.Printf("Message received on %s: %s\n", e.TopicPartition, string(e.Value))

			var transferReq handler.InitiatedTransfer
			json.Unmarshal(message, &transferReq)

			success := debitAccount(transferReq.SenderWalletID, transferReq.Amount)
			if success {
				log.Printf("Debitted %v", transferReq)
				// kafkaStream.ProduceMessage("transfer.debitted", string(message))
			}
		case kafka.Error:
			log.Printf("Error: %v\n", e)
		default:
			// Handle other events if needed
		}
	}

}

func debitAccount(sender int, amount float64) bool {
	// Simulate debit logic
	log.Printf("Debited %v from %v's account", amount, sender)
	return true
}
