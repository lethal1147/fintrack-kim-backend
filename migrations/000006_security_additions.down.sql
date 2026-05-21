ALTER TABLE sessions    DROP COLUMN IF EXISTS last_active_at;
DROP TABLE IF EXISTS totp_backup_codes;
ALTER TABLE users       DROP COLUMN IF EXISTS totp_enabled;
ALTER TABLE users       DROP COLUMN IF EXISTS totp_secret;
DROP TABLE IF EXISTS otp_tokens;
