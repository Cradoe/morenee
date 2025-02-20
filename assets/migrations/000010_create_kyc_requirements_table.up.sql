CREATE TABLE kyc_requirements (
    id SERIAL PRIMARY KEY,
    kyc_level_id INT NOT NULL,
    requirement VARCHAR(100) NOT NULL,
    FOREIGN KEY (kyc_level_id) REFERENCES kyc_levels(id) ON DELETE CASCADE       
);