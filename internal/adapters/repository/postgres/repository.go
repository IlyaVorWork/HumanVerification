package postgres

import (
	"context"
	"github.com/google/uuid"
	"humanVerification/internal/adapters/repository/postgres/human_verification"
)

type HumanVerificationRepository interface {
	CreateVerificationRequest(ctx context.Context, arg human_verification.CreateVerificationRequestParams) error
	ListVerificationRequestsByStatusesPaged(ctx context.Context, arg human_verification.ListVerificationRequestsByStatusesPagedParams) ([]human_verification.VerificationRequest, error)
	UpdateVerificationRequestStatus(ctx context.Context, arg human_verification.UpdateVerificationRequestStatusParams) error
	GetFileName(ctx context.Context, id uuid.UUID) (string, error)
	GetVerificationRequest(ctx context.Context, id uuid.UUID) (human_verification.VerificationRequest, error)
}
