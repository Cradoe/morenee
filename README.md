# Morenee API Documentation

## Overview
Morenee is a fintech system that provides secure and efficient financial transactions. This documentation covers the API design, system architecture, and key functionalities.

## Quick start

1. Clone the repository:

```bash
git clone https://github.com/cradoe/morenee.git
cd morenee
```

2. Create a `.env` file with the environment variables named in `compose.yml`:

- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`

3. Start the API, PostgreSQL, and Kafka:

```bash
make docker/up
```

The API runs on port 4444. Check it with `GET /health`.

To follow the app logs:

```bash
make docker/logs
```

To apply database migrations, set `DB_DSN` and run:

```bash
make migrations/up
```

## Tech Stack
The Morenee API backend is built using the following technologies:

- **Go**: Chosen for its high performance, concurrency support, and efficiency in building scalable systems.
- **Apache Kafka**: Used for event-driven architecture, ensuring reliability, scalability, and fault tolerance.
- **PostgreSQL**: A robust, high-performance relational database that supports complex queries and transactions.
- **Docker**: Used for containerization, ensuring consistency across environments and simplifying deployments.

## System Design & Architecture

Although Morenee is currently built as a monolithic application, it has been designed with scalability in mind. The system is structured to easily evolve into a larger distributed architecture when needed.

### Key Architectural Features:
- **Event-Driven Architecture**: Apache Kafka handles transaction processing asynchronously, ensuring system resilience and responsiveness.
- **Scalability Considerations**: The modular design allows services to be decoupled and independently scaled if required.
- **Database Transactions**: PostgreSQL ensures ACID compliance, providing consistency and reliability in financial transactions.
- **Background Workers**: Kafka consumers handle transaction finalization, ensuring fault tolerance and enabling retries or reversals in case of failure.
- **Containerization with Docker**: Docker ensures smooth deployment and consistency across different environments.

This architecture makes Morenee a solid foundation for a full-fledged distributed fintech system in the future.

## Available API Endpoints

### Health Check
- **GET /health** - Checks the status of the API.

### Authentication
- **POST /auth/login** - Logs in a user.
- **POST /auth/register** - Registers a new user.
- **POST /auth/verify-account** - Verifies a user account.
- **POST /auth/verify-account/resend** - Resends verification OTP.
- **POST /auth/forgot-password** - Initiates password reset.
- **POST /auth/reset-password** - Resets user password.

### Account Management
- **PATCH /account/pin** - Sets or updates account PIN.
- **GET /account/profile** - Fetches user profile.
- **PATCH /account/profile-picture** - Updates profile picture.
- **GET /account/next-of-kin** - Fetches next of kin details.
- **POST /account/next-of-kin** - Adds a next of kin.

### KYC (Know Your Customer)
- **POST /account/kyc/bvn** - Submits BVN for verification.
- **POST /account/kyc** - Submits KYC data.
- **GET /account/kyc** - Retrieves all user KYC data.
- **GET /kyc** - Retrieves KYC requirements.
- **GET /kyc/{id}** - Retrieves a specific KYC requirement.

### Wallet Management
- **GET /wallets** - Retrieves user wallets. A user can have multiple wallets. One is auto-generated at registration.
- **GET /wallets/{id}/details** - Fetches wallet details.
- **GET /wallets/{id}/balance** - Retrieves wallet balance.

### Transactions
- **POST /transactions/send-money** - Initiates a money transfer.
- **POST /transactions/withdraw** - Withdraws funds from a wallet.
- **POST /transactions/verify-pin** - Verifies a transaction PIN.
- **POST /transactions/confirm-transfer** - Confirms a money transfer.
- **POST /transactions/confirm-withdrawal** - Confirms a withdrawal.
- **GET /transactions/history** - Retrieves transaction history.
- **POST /transactions/transfer-fee** - Calculates transfer fees.
- **POST /transactions/topup** - Tops up a wallet.
- **POST /transactions/fund-transfer** - Funds a transfer.
- **POST /transactions/topup-details** - Retrieves top-up details.
- **POST /transactions/topup/verify** - Verifies a top-up.
- **POST /transactions/topup/confirm** - Confirms a top-up.
- **POST /transactions/virtual-accounts** - Retrieves virtual accounts.
- **POST /transactions/payment-link** - Generates a payment link.

### Notifications
- **GET /notifications** - Retrieves user notifications.
- **GET /notifications/{id}** - Retrieves a specific notification.

### Transaction Flow & Backend Logic
- **Initiating a Transaction**: When a user initiates a money transfer, the system first validates the transaction.
- **Kafka Event Publishing**: Once validated, the transaction event is published to a Kafka topic.
- **Worker Processing**: A Kafka consumer picks up the transaction event and processes it.
- **Database Updates**: After processing, the transaction details are stored in PostgreSQL.
- **Notification System**: A notification is generated and stored for the user to track the transaction status.

## Conclusion
Morenee is a robust fintech system designed to handle transactions efficiently and securely, with scalability in mind. By leveraging Go, Kafka, PostgreSQL, and Docker, it ensures high performance, reliability, and fault tolerance.
