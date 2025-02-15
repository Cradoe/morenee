package worker

import (
	"github.com/cradoe/morenee/internal/database"
	"github.com/cradoe/morenee/internal/stream"
)

type Worker struct {
	kafkaStream *stream.KafkaStream
	db          *database.DB
}

const (
	// transferDebitGroupID is used for workers that needs to take action whenever a request for debit was initiated
	transferDebitGroupID = "transfer-debit-group"

	// transferCreditGroupID is used for workers that needs to take action whenever a request for credit was initiated
	transferCreditGroupID = "transfer-credit-group"

	// transferFailed is used for workers that needs to take action whenever there's a failed transfer
	transferFailed = "transfer-failed-group"

	// transferSuccessGroupID is used for workers that needs to take action when a transfer recquest has been completed
	transferSuccessGroupID = "transfer-success-group"

	// Topics
	// transferDebitTopic is used to create request to debit the sender's wallet, when they initiate a transfer request to another user.
	transferDebitTopic = "transfer.debit"

	// transferCreditTopic is used to create request that credits the recipient's wallet during wallet-wallet transaction
	transferCreditTopic = "transfer.credit"

	// transferFailureTopic is used to create request to mark transaction as failed and revert all actions, to avoid inconsistent data
	transferFailureTopic = "transfer.failed"

	// transferSuccessTopic is used to create request to mark transaction as successful after debit and credit has been completed
	transferSuccessTopic = "transfer.success"
)

// Our workers typically needs access to database and kafka event stream
// worker-specific dependency can be passed as argument to the worker
func New(kafkaStream *stream.KafkaStream, db *database.DB) *Worker {
	return &Worker{
		kafkaStream: kafkaStream,
		db:          db,
	}
}
