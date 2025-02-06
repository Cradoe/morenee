ALTER TABLE transactions
ADD COLUMN reference_number VARCHAR(50) UNIQUE NOT NULL;
