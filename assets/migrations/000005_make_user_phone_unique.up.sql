ALTER TABLE users
ADD CONSTRAINT unique_phone_number UNIQUE (phone_number);