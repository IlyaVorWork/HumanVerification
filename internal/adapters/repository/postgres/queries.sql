-- ====================================================
-- HUMAN VERIFICATION
-- ====================================================

-- name: CreateVerificationRequest :exec
INSERT INTO verification_requests (id,
                                   event_id,
                                   correlation_id,
                                   apk_filename,
                                   status)
VALUES ($1, $2, $3, $4, 'pending');

-- name: ListVerificationRequestsByStatusesPaged :many
SELECT id,
       event_id,
       correlation_id,
       apk_filename,
       status,
       created_at,
       updated_at
FROM verification_requests
WHERE status = ANY(sqlc.arg(statuses)::text[])
ORDER BY created_at ASC
LIMIT $1 OFFSET $2;

-- name: UpdateVerificationRequestStatus :exec
UPDATE verification_requests
SET status     = $2,
    updated_at = NOW()
WHERE id = $1;

-- name: GetFileName :one
SELECT apk_filename FROM verification_requests
WHERE id = $1;

-- name: GetVerificationRequest :one
SELECT id, event_id, correlation_id, apk_filename, status, created_at, updated_at
FROM verification_requests
WHERE id = $1;
