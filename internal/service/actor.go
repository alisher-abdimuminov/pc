package service

import (
	"github.com/google/uuid"

	"pc/internal/audit"
)

func (a Actor) entry(action, entityType string, id *uuid.UUID, before, after any) audit.Entry {
	return audit.Entry{
		ActorID: a.ID, ActorRole: a.Role, Action: action, EntityType: entityType, EntityID: id,
		Before: before, After: after, IP: a.IP, UserAgent: a.UserAgent,
	}
}
