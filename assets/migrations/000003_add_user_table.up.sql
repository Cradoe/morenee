-- Create the users table
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),          
    first_name VARCHAR(100) NOT NULL,         
    last_name VARCHAR(100) NOT NULL,           
    phone_number VARCHAR(20) NOT NULL UNIQUE,                  
    pin VARCHAR(4),                       
    gender VARCHAR(6),                       
    email VARCHAR(255) NOT NULL UNIQUE,       
    status VARCHAR(20) DEFAULT 'active',      
    created_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    deleted_at TIMESTAMP,                     
    verified_at TIMESTAMP,                    
    hashed_password VARCHAR(60) NOT NULL      
);
