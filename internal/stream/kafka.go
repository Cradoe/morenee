package stream

import (
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

type KafkaStream struct {
	kafkaServers string
}

func New(kafkaServers string) *KafkaStream {
	return &KafkaStream{
		kafkaServers: kafkaServers,
	}
}

func (st *KafkaStream) ProduceMessage(topic, message string) error {
	producer, err := kafka.NewProducer(&kafka.ConfigMap{"bootstrap.servers": st.kafkaServers})
	if err != nil {
		return err
	}
	defer producer.Close()

	// Produce the message
	err = producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(message),
	}, nil)

	if err != nil {
		log.Printf("Failed to produce message: %v", err)
		return err
	}

	log.Printf("Message sent to topic %s", topic)
	return nil
}

type StreamConsumer struct {
	GroupId string
	Topic   string
}

func (st *KafkaStream) CreateConsumer(consumerStruct *StreamConsumer) (*kafka.Consumer, error) {
	consumer, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": st.kafkaServers,
		"group.id":          consumerStruct.GroupId,
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return nil, err
	}

	if err := consumer.Subscribe(consumerStruct.Topic, nil); err != nil {
		return nil, err
	}

	return consumer, nil
}
