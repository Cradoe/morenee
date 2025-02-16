CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),      
    sender_wallet_id UUID NOT NULL,                    
    recipient_wallet_id UUID NOT NULL,   
    reference_number VARCHAR(50) UNIQUE NOT NULL,                 
    amount DECIMAL(15, 2) NOT NULL,      
    description VARCHAR(100),
    status VARCHAR(20) DEFAULT 'pending',     
    created_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    FOREIGN KEY (sender_wallet_id) REFERENCES wallets(id) ON DELETE RESTRICT,
    FOREIGN KEY (recipient_wallet_id) REFERENCES wallets(id) ON DELETE RESTRICT
);