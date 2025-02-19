CREATE TABLE IF NOT EXISTS next_of_kins(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),        
    user_id UUID NOT NULL,                  
    email VARCHAR(255) NOT NULL,       
    first_name VARCHAR(100) NOT NULL,         
    last_name VARCHAR(100) NOT NULL,           
    phone_number VARCHAR(20) NOT NULL,                  
    address VARCHAR(255) NOT NULL,                  
    relationship VARCHAR(50) NOT NULL,                  
    created_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    deleted_at TIMESTAMP,    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE                 
)