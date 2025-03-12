# Morenee API Documentation

## Overview
Morenee is a fintech system that provides secure and efficient financial transactions. This documentation covers the API design, system architecture, and key functionalities.

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
- **GET /wallets** - Retrieves user wallets. A user can have multiple wallets. One is auto-generated after account verification in authentication.
- **GET /wallets/{id}/details** - Fetches wallet details.
- **GET /wallets/{id}/balance** - Retrieves wallet balance.

### Transactions
- **POST /transactions/send-money** - Initiates a money transfer.
- **GET /transactions/{id}** - Retrieves transaction details.
- **GET /transactions/wallet/{id}/transactions** - Lists all transactions for a specific wallet.

### Utilities
- **POST /utility/upload-file** - Uploads files.

### Error Handling
For all invalid routes, the system returns a `404 Not Found` error.

## Transaction Flow & Backend Logic

### Sending Money Flow:
1. **Pre-checks**: Validates sender's ability to send money and verifies balance sufficiency.
2. **Transaction Initiation**: Creates a pending transaction.
3. **Kafka Event Emission**: The transaction is published to Kafka for processing.
4. **Background Processing**:
   - Worker 1: Debits sender’s wallet.
   - Worker 2: Credits recipient’s wallet.
   - Worker 3: Finalizes transaction status.
5. **Failure Handling**: Automatic retries and reversals are in place to ensure consistency.

This design ensures high reliability and prevents data inconsistencies in financial transactions.

## Conclusion
Morenee is a robust fintech API designed to be scalable and reliable. While currently monolithic, its architecture is structured to scale into a fully distributed system as needed. By using Go, Kafka, PostgreSQL, and Docker, Morenee ensures security, efficiency, and resilience in financial transactions.

