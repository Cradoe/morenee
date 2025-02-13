CREATE TABLE IF NOT EXISTS transaction_logs (
    id SERIAL PRIMARY KEY,                   
    transaction_id INT NOT NULL,                    
    user_id INT NOT NULL,          
    action VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT
);
