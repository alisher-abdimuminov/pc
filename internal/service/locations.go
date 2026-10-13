package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"pc/internal/apperr"
	"pc/internal/audit"
	"pc/internal/db"
	"pc/internal/geo"
)

type LocationService struct{ pool *pgxpool.Pool }

func NewLocationService(pool *pgxpool.Pool) *LocationService { return &LocationService{pool: pool} }

type LocationInput struct {
	Name            string   `json:"name"`
	Points          []string `json:"points"` // "lat,lng"
	ToleranceMeters int32    `json:"tolerance_meters"`
	LocationURL     string   `json:"location_url"`
}

// validate parses and checks the input, returning the name and JSONB polygon.
func (in LocationInput) validate() (string, []byte, error) {
	fields := map[string]string{}
	name, err := validateName(in.Name)
	if err != nil {
		fields["name"] = "Majburiy maydon"
	}
	var polygon []byte
	pts, err := geo.ParsePoints(in.Points)
	var pe *geo.PointError
	switch {
	case errors.As(err, &pe):
		fields["points"] = strconv.Itoa(pe.Index+1) + "-nuqta noto'g'ri. Format: 39.667365,66.878348"
	case errors.Is(err, geo.ErrTooFewPoints):
		fields["points"] = "Kamida 3 ta nuqta kiriting"
	case err != nil:
		fields["points"] = "Nuqtalar noto'g'ri"
	default:
		polygon, _ = json.Marshal(pts)
	}
	if in.ToleranceMeters < 0 || in.ToleranceMeters > 1000 {
		fields["tolerance_meters"] = "0 dan 1000 gacha bo'lishi kerak"
	}
	if len(in.LocationURL) > 500 {
		fields["location_url"] = "Havola juda uzun"
	}
	if len(fields) > 0 {
		return "", nil, apperr.Validation("Ma'lumotlar noto'g'ri", fields)
	}
	return name, polygon, nil
}

func (s *LocationService) List(ctx context.Context, p ListParams) (Page[LocationDTO], error) {
	p = p.Norm()
	q := db.New(s.pool)
	rows, err := q.ListLocations(ctx, db.ListLocationsParams{Search: p.Search, Lim: p.Limit(), Off: p.Offset()})
	if err != nil {
		return Page[LocationDTO]{}, err
	}
	total, err := q.CountLocations(ctx, p.Search)
	if err != nil {
		return Page[LocationDTO]{}, err
	}
	items := make([]LocationDTO, len(rows))
	for i, r := range rows {
		items[i] = locationDTO(r)
	}
	return newPage(items, total, p), nil
}

func (s *LocationService) Get(ctx context.Context, id uuid.UUID) (LocationDTO, error) {
	l, err := db.New(s.pool).GetLocation(ctx, id)
	if err != nil {
		return LocationDTO{}, notFoundOr(err)
	}
	return locationDTO(l), nil
}

func (s *LocationService) Create(ctx context.Context, actor Actor, in LocationInput) (LocationDTO, error) {
	name, polygon, err := in.validate()
	if err != nil {
		return LocationDTO{}, err
	}
	var out LocationDTO
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		l, err := q.CreateLocation(ctx, db.CreateLocationParams{Name: name, Polygon: polygon, ToleranceMeters: in.ToleranceMeters, LocationUrl: in.LocationURL})
		if err != nil {
			return err
		}
		out = locationDTO(l)
		return audit.Write(ctx, tx, actor.entry(audit.ActionCreate, "location", &l.ID, nil, out))
	})
	return out, mapDBError(err)
}

func (s *LocationService) Update(ctx context.Context, actor Actor, id uuid.UUID, in LocationInput) (LocationDTO, error) {
	name, polygon, err := in.validate()
	if err != nil {
		return LocationDTO{}, err
	}
	var out LocationDTO
	err = InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := q.GetLocation(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		l, err := q.UpdateLocation(ctx, db.UpdateLocationParams{ID: id, Name: name, Polygon: polygon, ToleranceMeters: in.ToleranceMeters, LocationUrl: in.LocationURL})
		if err != nil {
			return err
		}
		out = locationDTO(l)
		return audit.Write(ctx, tx, actor.entry(audit.ActionUpdate, "location", &id, locationDTO(old), out))
	})
	return out, mapDBError(err)
}

func (s *LocationService) Delete(ctx context.Context, actor Actor, id uuid.UUID) error {
	err := InTx(ctx, s.pool, func(tx pgx.Tx, q *db.Queries) error {
		old, err := q.GetLocation(ctx, id)
		if err != nil {
			return notFoundOr(err)
		}
		if err := q.DeleteLocation(ctx, id); err != nil {
			return err
		}
		return audit.Write(ctx, tx, actor.entry(audit.ActionDelete, "location", &id, locationDTO(old), nil))
	})
	return mapDBError(err)
}
