ALTER TABLE users
ADD CONSTRAINT IF NOT EXISTS unique_phone_number UNIQUE (phone_number);