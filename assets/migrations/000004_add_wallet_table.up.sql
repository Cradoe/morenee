CREATE TABLE IF NOT EXISTS wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),            
    user_id UUID NOT NULL,                    
    balance DECIMAL(15, 2) DEFAULT 0.00,      
    account_number VARCHAR(10) NOT NULL UNIQUE,       
    currency VARCHAR(10) DEFAULT 'NGN',    
    single_transfer_limit DECIMAL(15,2) NOT NULL DEFAULT 0.00,   
    daily_transfer_limit DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    max_balance DECIMAL(15,2) NOT NULL DEFAULT 0.00,
    status VARCHAR(20) DEFAULT 'active',     
    created_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    deleted_at TIMESTAMP, 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE 
);
