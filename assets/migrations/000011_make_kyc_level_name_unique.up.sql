ALTER TABLE kyc_levels
ADD CONSTRAINT unique_level_name UNIQUE (level_name);
