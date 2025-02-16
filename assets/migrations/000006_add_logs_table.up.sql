CREATE TABLE IF NOT EXISTS activity_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),    
    user_id UUID NOT NULL,          
    entity_id UUID NOT NULL,                    
    entity VARCHAR(50) NOT NULL,
    description VARCHAR(100) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(), 
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
