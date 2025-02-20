CREATE TABLE IF NOT EXISTS kyc_levels (
    id SERIAL PRIMARY KEY,
    level_name VARCHAR(50) NOT NULL, 
    single_transfer_limit DECIMAL(15, 2) NOT NULL, 
    daily_transfer_limit DECIMAL(15, 2) NOT NULL,
    wallet_balance_limit DECIMAL(15, 2) NOT NULL
);