package service

import "github.com/google/uuid"

// Actor identifies who performs an action, for auditing.
type Actor struct {
	ID        *uuid.UUID
	Role      string
	IP        string
	UserAgent string
}
