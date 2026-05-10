package service

import "errors"

const (
	StatusPending    = "pending"
	StatusInProgress = "in_progress"
	StatusApproved   = "approved"
	StatusRejected   = "rejected"
)

var ErrInvalidTransition = errors.New("invalid status transition")

var validTransitions = map[string][]string{
	StatusPending:    {StatusInProgress},
	StatusInProgress: {StatusApproved, StatusRejected},
}

func isValidTransition(from, to string) bool {
	for _, allowed := range validTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
