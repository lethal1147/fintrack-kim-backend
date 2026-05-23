ALTER TABLE otp_tokens
    DROP CONSTRAINT otp_tokens_purpose_check,
    ADD CONSTRAINT otp_tokens_purpose_check
        CHECK (purpose IN ('password_change', 'password_reset'));
