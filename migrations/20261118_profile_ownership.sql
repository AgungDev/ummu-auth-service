-- Run only after the legacy full_name values have been imported to Profile Service
-- and the imported profiles have been verified.
ALTER TABLE users DROP COLUMN IF EXISTS full_name;
