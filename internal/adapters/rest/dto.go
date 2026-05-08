package rest

import "humanVerification/internal/adapters/repository/postgres/human_verification"

type GetRequestsListByStatusesOutDTO struct {
	Items []human_verification.VerificationRequest `json:"items"`
	Page  int                                      `json:"page"`
	Size  int                                      `json:"size"`
}

type UpdateRequestStatusInDTO struct {
	Status string `json:"status" binding:"required" example:"approved" enums:"approved,rejected"`
}

type RequestFileLinkOutDTO struct {
	Link string `json:"link" example:"https://example.com/file.apk"`
}

type ErrorResponseDTO struct {
	Error string `json:"error" example:"invalid request"`
}
