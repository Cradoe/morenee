package worker

import (
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/stream"
)

type Worker struct {
	kafkaStream *stream.KafkaStream
	db          *database.DB
}

// create all possible groupdId and topics
const (
	transferDebitGroupID   = "transfer-debit-group"
	transferCreditGroupID  = "transfer-credit-group"
	transferFailed         = "transfer-failed-group"
	transferSuccessGroupID = "transfer-success-group"

	transferCreatedTopic = "transfer.created"
	transferDebitTopic   = "transfer.debit"
	transferCreditTopic  = "transfer.credit"
	transferFailureTopic = "transfer.failed"
	transferSuccessTopic = "transfer.success"
)

func New(kafkaStream *stream.KafkaStream, db *database.DB) *Worker {
	return &Worker{
		kafkaStream: kafkaStream,
		db:          db,
	}
}
