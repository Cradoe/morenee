-- Create the users table
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,                    
    first_name VARCHAR(255) NOT NULL,         
    last_name VARCHAR(255) NOT NULL,           
    phone_number VARCHAR(20) NOT NULL,                  
    gender VARCHAR(10),                       
    email VARCHAR(255) NOT NULL UNIQUE,       
    status VARCHAR(20) DEFAULT 'active',      
    created_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    deleted_at TIMESTAMP,                     
    verified_at TIMESTAMP,                    
    hashed_password VARCHAR(255) NOT NULL      
);
