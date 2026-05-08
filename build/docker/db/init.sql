CREATE TABLE verification_requests
(
    id             UUID PRIMARY KEY,
    event_id       UUID unique NOT NULL,
    correlation_id UUID unique NOT NULL,
    apk_filename   VARCHAR     NOT NULL,
    status         varchar     NOT NULL,
    created_at     TIMESTAMP   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP   NOT NULL DEFAULT NOW()
);

ALTER TABLE verification_requests
    ADD CONSTRAINT verification_request_status_check
        CHECK (status IN (
                          'pending',
                          'in_progress',
                          'approved',
                          'rejected'
            ));