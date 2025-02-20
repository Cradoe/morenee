CREATE TABLE user_kyc_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),  
    user_id UUID NOT NULL,
    kyc_requirement_id UUID NOT NULL,
    submission_data TEXT NOT NULL,
    verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP, 
    UNIQUE (user_id, kyc_requirement_id), -- Ensures no duplicate submissions per user and requirement
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (kyc_requirement_id) REFERENCES kyc_requirements(id) ON DELETE RESTRICT
);
