CREATE TABLE IF NOT EXISTS account_logs (
    id SERIAL PRIMARY KEY,                   
    user_id INT NOT NULL,          
    type_id INT NOT NULL,                    
    type VARCHAR(50) NOT NULL,
    description VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
);
