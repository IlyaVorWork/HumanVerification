package rest

import (
	"context"
	"database/sql"
	"errors"
	"github.com/gin-gonic/gin"
	"humanVerification/internal/adapters/kafka"
	"humanVerification/internal/adapters/repository/postgres/human_verification"
	"net/http"
	"strconv"
	"strings"
)

const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusApproved   = "approved"
	StatusRejected   = "rejected"
)

type Service interface {
	GetVerificationRequests(page, size int, statuses []string) ([]human_verification.VerificationRequest, error)
	CreateVerificationRequest(event kafka.Event) error
	UpdateVerificationRequestStatus(requestId, status string) error
	GetFileLink(requestId string) (string, error)
}

type VerificationRequestHandler struct {
	service Service
}

func NewVerificationRequestHandler(service Service) *VerificationRequestHandler {
	return &VerificationRequestHandler{service: service}
}

// GetRequestsListByStatuses godoc
// @Summary Список запросов на верификацию
// @Description Возвращает список запросов, у которых статус совпадает хотя бы с одним из переданных значений.
// @Tags verification-requests
// @Accept json
// @Produce json
// @Param page query int false "Номер страницы, начиная с 0"
// @Param size query int false "Размер страницы"
// @Param statuses query []string false "Статусы для фильтрации"
// @Param status query string false "Статусы через запятую, если не передан statuses"
// @Success 200 {object} GetRequestsListByStatusesOutDTO
// @Failure 400 {object} ErrorResponseDTO
// @Failure 500 {object} ErrorResponseDTO
// @Router /verifications [get]
func (h *VerificationRequestHandler) GetRequestsListByStatuses(c *gin.Context) {
	page, err := parsePositiveIntQuery(c, "page", 0)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	size, err := parsePositiveIntQuery(c, "size", 50)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	statuses := parseStatusesQuery(c)
	if len(statuses) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one status is required"})
		return
	}

	requests, err := h.service.GetVerificationRequests(page*size, size, statuses)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, GetRequestsListByStatusesOutDTO{
		Items: requests,
		Page:  page,
		Size:  size,
	})
}

// UpdateRequestStatus godoc
// @Summary Обновить статус запроса
// @Description Меняет статус запроса на approved или rejected.
// @Tags verification-requests
// @Accept json
// @Produce json
// @Param requestId path string true "Идентификатор запроса"
// @Param body body UpdateRequestStatusInDTO true "Новый статус запроса"
// @Success 204 {string} string "No Content"
// @Failure 400 {object} ErrorResponseDTO
// @Failure 404 {object} ErrorResponseDTO
// @Failure 500 {object} ErrorResponseDTO
// @Router /verifications/{requestId} [patch]
func (h *VerificationRequestHandler) UpdateRequestStatus(c *gin.Context) {
	requestID := c.Param("requestId")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "requestId is required"})
		return
	}

	var body UpdateRequestStatusInDTO
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if body.Status != StatusApproved && body.Status != StatusRejected {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be approved or rejected"})
		return
	}

	if err := h.service.UpdateVerificationRequestStatus(requestID, body.Status); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetRequestFileLink godoc
// @Summary Получить ссылку на файл
// @Description Возвращает ссылку на APK-файл, связанный с указанным запросом.
// @Tags verification-requests
// @Accept json
// @Produce json
// @Param requestId path string true "Идентификатор запроса"
// @Success 200 {object} RequestFileLinkOutDTO
// @Failure 400 {object} ErrorResponseDTO
// @Failure 404 {object} ErrorResponseDTO
// @Failure 500 {object} ErrorResponseDTO
// @Router /verifications/{requestId}/link [get]
func (h *VerificationRequestHandler) GetRequestFileLink(c *gin.Context) {
	requestID := c.Param("requestId")
	if requestID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "requestId is required"})
		return
	}

	link, err := h.service.GetFileLink(requestID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, RequestFileLinkOutDTO{Link: link})
}

func parsePositiveIntQuery(c *gin.Context, key string, fallback int) (int, error) {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return 0, errors.New("invalid " + key)
	}

	return value, nil
}

func parseStatusesQuery(c *gin.Context) []string {
	statuses := c.QueryArray("statuses")
	if len(statuses) == 0 {
		raw := strings.TrimSpace(c.Query("status"))
		if raw != "" {
			statuses = strings.Split(raw, ",")
		}
	}

	out := make([]string, 0, len(statuses))
	seen := make(map[string]struct{}, len(statuses))
	for _, status := range statuses {
		status = strings.TrimSpace(status)
		if status == "" {
			continue
		}
		if _, ok := seen[status]; ok {
			continue
		}
		seen[status] = struct{}{}
		out = append(out, status)
	}
	return out
}

func writeError(c *gin.Context, err error) {
	if errors.Is(err, context.Canceled) {
		c.JSON(http.StatusRequestTimeout, gin.H{"error": "request canceled"})
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}
