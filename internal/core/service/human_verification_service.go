package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"humanVerification/internal/adapters/kafka"
	"humanVerification/internal/adapters/repository/postgres"
	"humanVerification/internal/adapters/repository/postgres/human_verification"
	"humanVerification/internal/adapters/s3"
)

type Service struct {
	repo     postgres.HumanVerificationRepository
	producer *kafka.Producer
	fs       s3.FileStorage
}

func NewService(repo postgres.HumanVerificationRepository, producer *kafka.Producer, fs s3.FileStorage) *Service {
	return &Service{repo: repo, producer: producer, fs: fs}
}

func (s *Service) GetVerificationRequests(page, size int, statuses []string) ([]human_verification.VerificationRequest, error) {
	ctx := context.Background()

	list, err := s.repo.ListVerificationRequestsByStatusesPaged(ctx, human_verification.ListVerificationRequestsByStatusesPagedParams{
		Limit:    int32(size),
		Offset:   int32(page),
		Statuses: statuses,
	})

	if err != nil {
		return nil, err
	}

	if list == nil {
		return []human_verification.VerificationRequest{}, nil
	}

	return list, nil
}

func (s *Service) CreateVerificationRequest(event kafka.Event) error {
	ctx := context.Background()

	err := s.repo.CreateVerificationRequest(ctx, human_verification.CreateVerificationRequestParams{
		ID:            uuid.New(),
		EventID:       uuid.MustParse(event.EventID),
		CorrelationID: uuid.MustParse(event.CorrelationID),
		ApkFilename:   event.FileName,
	})

	if err != nil {
		return err
	}

	return nil
}

func (s *Service) UpdateVerificationRequestStatus(requestId, status string) error {
	ctx := context.Background()

	id := uuid.MustParse(requestId)

	current, err := s.repo.GetVerificationRequest(ctx, id)
	if err != nil {
		return err
	}

	if !isValidTransition(current.Status, status) {
		return ErrInvalidTransition
	}

	if err := s.repo.UpdateVerificationRequestStatus(ctx, human_verification.UpdateVerificationRequestStatusParams{
		ID:     id,
		Status: status,
	}); err != nil {
		return err
	}

	if status == StatusInProgress {
		return nil
	}

	message := kafka.HumanVerifyFailed
	if status == StatusApproved {
		message = kafka.HumanVerifySucceeded
	}

	return s.producer.Send(ctx, kafka.TopicVerificationResponses, current.CorrelationID.String(), kafka.Event{
		EventID:       uuid.New().String(),
		CorrelationID: current.CorrelationID.String(),
		Type:          message,
		FileName:      current.ApkFilename,
		Timestamp:     time.Now(),
	})
}

func (s *Service) GetFileLink(requestId string) (string, error) {
	ctx := context.Background()

	fileName, err := s.repo.GetFileName(ctx, uuid.MustParse(requestId))
	if err != nil {
		return "", err
	}

	link, err := s.fs.GetFileLink(fileName)
	if err != nil {
		return "", err
	}

	return link, nil
}
