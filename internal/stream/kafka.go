package stream

import (
	"context"
	"log"
	"strings"

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

// EnsureTopicsExist ensures that all the topics in the provided list exist in Kafka.
// If a topic does not exist, it will be created with the specified number of partitions and replication factor.
// The function will be called on startup, when New constructor is called.
// This approach minimizes unnecessary topic creation requests, improving performance and reducing potential Kafka errors.
//
// Why: Kafka does not automatically create topics by default for production environments. This function ensures
// that missing topics are created ahead of time to avoid runtime errors when producing or consuming messages.
func (st *KafkaStream) EnsureTopicsExist(topics []string) error {
	// Create an admin client to interact with Kafka for administrative tasks (e.g., topic creation).
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": st.kafkaServers})
	if err != nil {
		return err
	}
	defer adminClient.Close()

	// Fetch metadata for all topics to check which topics already exist.
	metadata, err := adminClient.GetMetadata(nil, false, 5000)
	if err != nil {
		return err
	}

	// Determine the number of brokers by splitting the bootstrap servers by commas.
	numBrokers := len(strings.Split(st.kafkaServers, ","))

	// missing topics that need to be created.
	missingTopics := []kafka.TopicSpecification{}

	// Iterate through the list of requested topics.
	for _, topic := range topics {
		// Check if the topic already exists in the metadata.
		if _, exists := metadata.Topics[topic]; !exists {
			// If the topic is not found in metadata, add it to the list of topics to be created.
			missingTopics = append(missingTopics, kafka.TopicSpecification{
				Topic:             topic,
				NumPartitions:     1,
				ReplicationFactor: int(numBrokers), // Use the number of brokers as the replication factor.
			})
		}
	}

	// If there are any missing topics, create them using the admin client.
	if len(missingTopics) > 0 {
		_, err = adminClient.CreateTopics(context.Background(), missingTopics)
		if err != nil {
			return err
		}
	}

	return nil
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

func (st *KafkaStream) EnsureTopicExists(topic string) error {
	adminClient, err := kafka.NewAdminClient(&kafka.ConfigMap{"bootstrap.servers": st.kafkaServers})
	if err != nil {
		return err
	}
	defer adminClient.Close()

	_, err = adminClient.CreateTopics(
		context.Background(),
		[]kafka.TopicSpecification{
			{Topic: topic, NumPartitions: 1, ReplicationFactor: 1},
		},
	)
	if err != nil {
		return err
	}

	return nil
}
