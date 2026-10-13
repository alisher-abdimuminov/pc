package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/db"
)

type AuditService struct {
	pool *pgxpool.Pool
	loc  *time.Location
}

func NewAuditService(pool *pgxpool.Pool, loc *time.Location) *AuditService {
	return &AuditService{pool: pool, loc: loc}
}

type AuditFilter struct {
	ActorID    *uuid.UUID
	Action     string
	EntityType string
	From, To   string // YYYY-MM-DD, inclusive, in the business timezone
}

func (s *AuditService) dayStart(v string, plusDays int) (pgtype.Timestamptz, error) {
	if v == "" {
		return pgtype.Timestamptz{}, nil
	}
	t, err := time.ParseInLocation(DateLayout, v, s.loc)
	if err != nil {
		return pgtype.Timestamptz{}, err
	}
	return pgtype.Timestamptz{Time: t.AddDate(0, 0, plusDays), Valid: true}, nil
}

func auditDTO(a db.AuditLog, username, name pgtype.Text) AuditDTO {
	d := AuditDTO{ID: a.ID, ActorID: uuidPtr(a.ActorID), ActorRole: a.ActorRole, ActorUsername: textPtr(username),
		ActorName: textPtr(name), Action: a.Action, EntityType: a.EntityType, EntityID: uuidPtr(a.EntityID),
		Changes: a.Changes, UserAgent: a.UserAgent, CreatedAt: a.CreatedAt.Time}
	if a.Ip != nil {
		d.IP = a.Ip.String()
	}
	return d
}

func (s *AuditService) List(ctx context.Context, p ListParams, f AuditFilter) (Page[AuditDTO], error) {
	p = p.Norm()
	from, err := s.dayStart(f.From, 0)
	if err != nil {
		return Page[AuditDTO]{}, fieldErr("from", "Sana noto'g'ri (YYYY-MM-DD)")
	}
	to, err := s.dayStart(f.To, 1)
	if err != nil {
		return Page[AuditDTO]{}, fieldErr("to", "Sana noto'g'ri (YYYY-MM-DD)")
	}
	q := db.New(s.pool)
	rows, err := q.ListAuditLogs(ctx, db.ListAuditLogsParams{
		ActorID: nullUUID(f.ActorID), Action: f.Action, EntityType: f.EntityType, FromTs: from, ToTs: to,
		Lim: p.Limit(), Off: p.Offset(),
	})
	if err != nil {
		return Page[AuditDTO]{}, err
	}
	total, err := q.CountAuditLogs(ctx, db.CountAuditLogsParams{
		ActorID: nullUUID(f.ActorID), Action: f.Action, EntityType: f.EntityType, FromTs: from, ToTs: to,
	})
	if err != nil {
		return Page[AuditDTO]{}, err
	}
	items := make([]AuditDTO, len(rows))
	for i, r := range rows {
		items[i] = auditDTO(db.AuditLog{ID: r.ID, ActorID: r.ActorID, ActorRole: r.ActorRole, Action: r.Action,
			EntityType: r.EntityType, EntityID: r.EntityID, Changes: r.Changes, Ip: r.Ip, UserAgent: r.UserAgent,
			CreatedAt: r.CreatedAt}, r.ActorUsername, r.ActorName)
	}
	return newPage(items, total, p), nil
}

func (s *AuditService) Get(ctx context.Context, id uuid.UUID) (AuditDTO, error) {
	r, err := db.New(s.pool).GetAuditLog(ctx, id)
	if err != nil {
		return AuditDTO{}, notFoundOr(err)
	}
	return auditDTO(db.AuditLog{ID: r.ID, ActorID: r.ActorID, ActorRole: r.ActorRole, Action: r.Action,
		EntityType: r.EntityType, EntityID: r.EntityID, Changes: r.Changes, Ip: r.Ip, UserAgent: r.UserAgent,
		CreatedAt: r.CreatedAt}, r.ActorUsername, r.ActorName), nil
}
