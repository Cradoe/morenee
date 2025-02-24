package worker

import (
	"context"

	"github.com/cradoe/morenee/internal/helper"
	"github.com/cradoe/morenee/internal/repository"
	"github.com/cradoe/morenee/internal/smtp"
	"github.com/cradoe/morenee/internal/stream"
)

type Worker struct {
	UserRepo        repository.UserRepository
	TransactionRepo repository.TransactionRepository
	WalletRepo      repository.WalletRepository
	KycRepo         repository.KycRepository
	ActivityRepo    repository.ActivityRepository

	KafkaStream *stream.KafkaStream
	Ctx         context.Context
	Helper      *helper.Helper
	Mailer      *smtp.Mailer
}

const (
	// transferDebitGroupID is used for workers that needs to take action whenever a request for debit was initiated
	transferDebitGroupID = "transfer-debit-group"

	// transferCreditGroupID is used for workers that needs to take action whenever a request for credit was initiated
	transferCreditGroupID = "transfer-credit-group"

	// transferSuccessGroupID is used for workers that needs to take action when a transfer recquest has been completed
	transferSuccessGroupID = "transfer-success-group"

	// Topics
	// TransferDebitTopic is used to create request to debit the sender's wallet, when they initiate a transfer request to another user.
	TransferDebitTopic = "transfer.debit"

	// TransferCreditTopic is used to create request that credits the recipient's wallet during wallet-wallet transaction
	TransferCreditTopic = "transfer.credit"

	// TransferFailureTopic is used to create request to mark transaction as failed and revert all actions, to avoid inconsistent data
	TransferFailureTopic = "transfer.failed"

	// TransferSuccessTopic is used to create request to mark transaction as successful after debit and credit has been completed
	TransferSuccessTopic = "transfer.success"
)

// Our workers typically needs access to database and kafka event stream
// worker-specific dependency can be passed as argument to the worker
func New(wk *Worker) *Worker {
	return &Worker{
		UserRepo:        wk.UserRepo,
		TransactionRepo: wk.TransactionRepo,
		WalletRepo:      wk.WalletRepo,
		KycRepo:         wk.KycRepo,
		ActivityRepo:    wk.ActivityRepo,

		KafkaStream: wk.KafkaStream,
		Ctx:         wk.Ctx,
		Helper:      wk.Helper,
		Mailer:      wk.Mailer,
	}
}
