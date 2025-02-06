CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,                   
    sender_wallet_id INT NOT NULL,                    
    recipient_wallet_id INT NOT NULL,                    
    amount DECIMAL(15, 2) NOT NULL,      
    status VARCHAR(20) DEFAULT 'pending',     
    created_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    FOREIGN KEY (sender_wallet_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (recipient_wallet_id) REFERENCES users(id) ON DELETE RESTRICT
);