// Successful transactions are the ones that as gone through debitting (sender) and creditting (recipient)
// A record was created in the transactions table synchronousely when the transfer was initiated
// We need to mark that record as successful.
// We also need to send necessary notifications to both involed users.
package worker

import (
	"encoding/json"
	"log"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/cradoe/morenee/internal/handler"
	"github.com/cradoe/morenee/internal/repository"
	"github.com/cradoe/morenee/internal/stream"
)

func (wk *Worker) SuccessTransferWorker() {
	consumer, err := wk.KafkaStream.CreateConsumer(&stream.StreamConsumer{
		GroupId: transferSuccessGroupID,
		Topic:   TransferSuccessTopic,
	})

	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}
	defer consumer.Close() // Ensure cleanup

	for {
		select {
		case <-wk.Ctx.Done():
			log.Println("SuccessTransferWorker received cancellation signal, shutting down...")
			return
		default:
			// Poll for Kafka events
			event := consumer.Poll(100)
			switch e := event.(type) {
			case *kafka.Message:
				message := e.Value
				var transferReq *handler.TransactionResponseData
				json.Unmarshal(message, &transferReq)

				success := wk.completeTransferOperation(transferReq)
				if success {
					// Send notifications to the sender and receiver
					log.Printf("Transfer completed successfully: %v", transferReq)
					wk.sendTransactionAlerts(transferReq)
				}
			case kafka.Error:
				log.Printf("Error: %v\n", e)
			case *kafka.AssignedPartitions:
				consumer.Assign(e.Partitions)
			case *kafka.RevokedPartitions:
				consumer.Unassign()
			}
		}
	}
}

func (wk *Worker) completeTransferOperation(transferReq *handler.TransactionResponseData) bool {
	_, err := wk.TransactionRepo.UpdateStatus(transferReq.ID, repository.TransactionStatusCompleted)
	if err != nil {
		log.Printf("Error updating transaction status: %v", err)
		return false
	}

	wk.Helper.BackgroundTask(nil, func() error {
		_, err = wk.ActivityRepo.Insert(&repository.ActivityLog{
			UserID:      transferReq.Sender.ID,
			Entity:      repository.ActivityLogTransactionEntity,
			EntityId:    transferReq.ID,
			Description: handler.TransactionActivityLogSuccessDescription,
		})

		if err != nil {
			log.Printf("Error logging successful transacton action: %v", err)
			return err
		}
		return nil
	})

	return true
}

func (wk *Worker) sendTransactionAlerts(transferReq *handler.TransactionResponseData) bool {

	sender, _, err := wk.UserRepo.GetOne(transferReq.Sender.ID)
	if err != nil {
		log.Printf("Error finding sender's account for debit alert: %v", err)
		return false
	}

	recipient, _, err := wk.UserRepo.GetOne(transferReq.Recipient.ID)
	if err != nil {
		log.Printf("Error finding recipient's account for debit alert: %v", err)
		return false
	}

	senderWallet, _, err := wk.WalletRepo.GetOne(transferReq.Sender.Wallet.ID)
	if err != nil {
		log.Printf("Error finding sender's wallet for debit alert: %v", err)
		return false
	}

	recipientWallet, _, err := wk.WalletRepo.GetOne(transferReq.Recipient.Wallet.ID)
	if err != nil {
		log.Printf("Error finding sender's wallet for debit alert: %v", err)
		return false
	}

	// debit alert to sender
	wk.Helper.BackgroundTask(nil, func() error {
		emailData := wk.Helper.NewEmailData()
		emailData["Name"] = sender.FirstName + " " + sender.LastName
		emailData["BankName"] = transferReq.Sender.Wallet.BankName
		emailData["Amount"] = transferReq.Amount
		emailData["RecipientName"] = recipient.FirstName + " " + recipient.LastName
		emailData["RecipientAccountNumber"] = recipientWallet.AccountNumber
		emailData["TransactionID"] = transferReq.ReferenceNumber
		emailData["NewBalance"] = senderWallet.Balance

		err = wk.Mailer.Send(sender.Email, emailData, "debit-alert.tmpl")
		if err != nil {
			log.Printf("Error sending debit email alert: %v", err)
			return err
		}

		return nil
	})

	// credit alert to recipient
	wk.Helper.BackgroundTask(nil, func() error {
		emailData := wk.Helper.NewEmailData()
		emailData["Name"] = recipient.FirstName + " " + recipient.LastName
		emailData["BankName"] = transferReq.Recipient.Wallet.BankName
		emailData["Amount"] = transferReq.Amount
		emailData["SenderName"] = sender.FirstName + " " + sender.LastName
		emailData["SenderAccountNumber"] = senderWallet.AccountNumber
		emailData["TransactionID"] = transferReq.ReferenceNumber
		emailData["NewBalance"] = recipientWallet.Balance

		err = wk.Mailer.Send(recipient.Email, emailData, "credit-alert.tmpl")
		if err != nil {
			log.Printf("Error sending credit email alert: %v", err)
			return err
		}

		return nil
	})

	return true
}
