// Package inventory lists a dealership's vehicles and flags the aging ones.
package inventory

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/actions"
	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/catalog"
)

// ErrNotFound means the vehicle does not exist or belongs to another dealership.
var ErrNotFound = errors.New("vehicle not found")

// Sort orders. Ties are always broken by vehicle id so pages never overlap.
const (
	SortAgeDesc   = "age_desc"
	SortAgeAsc    = "age_asc"
	SortPriceDesc = "price_desc"
	SortPriceAsc  = "price_asc"
)

// orderBy maps a Sort to SQL. The map keeps user input out of the query text.
var orderBy = map[string]string{
	SortAgeDesc:   "v.stocked_at ASC",
	SortAgeAsc:    "v.stocked_at DESC",
	SortPriceDesc: "v.price DESC",
	SortPriceAsc:  "v.price ASC",
}

// Vehicle is a stocked vehicle with its derived age and latest action.
type Vehicle struct {
	ID            string           `json:"id"`
	VIN           string           `json:"vin"`
	Make          catalog.NamedRef `json:"make"`
	Model         catalog.NamedRef `json:"model"`
	Year          int              `json:"year"`
	Price         string           `json:"price"`
	StockedAt     string           `json:"stocked_at"`
	AgeDays       int              `json:"age_days"`
	IsAging       bool             `json:"is_aging"`
	CurrentAction *actions.Action  `json:"current_action"`
}

// Filter selects and orders vehicles. Nil pointers and empty strings mean "no filter".
type Filter struct {
	MakeID     string
	ModelID    string
	Aging      *bool
	MinAgeDays *int
	MaxAgeDays *int
	Sort       string
	Page       int
	PageSize   int
}

// Store reads vehicles. Every method is scoped to a dealership.
type Store interface {
	// List returns one page of matching vehicles and the total number of matches.
	List(ctx context.Context, dealershipID string, f Filter, thresholdDays int) ([]Vehicle, int, error)
	// Get returns one vehicle, or ErrNotFound.
	Get(ctx context.Context, dealershipID, id string, thresholdDays int) (Vehicle, error)
	// Counts returns how many vehicles the dealership stocks and how many are aging.
	Counts(ctx context.Context, dealershipID string, thresholdDays int) (total, aging int, err error)
}

// PGStore is the PostgreSQL implementation of Store.
type PGStore struct{ db *pgxpool.Pool }

// NewPGStore returns a Store backed by db.
func NewPGStore(db *pgxpool.Pool) *PGStore { return &PGStore{db: db} }

// vehicleSelect has one %s: the placeholder holding the aging threshold, such as "$2::int".
const vehicleSelect = `
	SELECT v.id::text, v.vin, k.id::text, k.name, m.id::text, m.name, v.year, v.price::text,
	       v.stocked_at, CURRENT_DATE - v.stocked_at, v.stocked_at < CURRENT_DATE - %s,
	       a.id::text, a.action_type, a.note, a.created_by, a.created_at
	FROM vehicles v
	JOIN models m ON m.id = v.model_id
	JOIN makes k ON k.id = m.make_id
	LEFT JOIN LATERAL (
		SELECT va.id, va.action_type, va.note, u.username AS created_by, va.created_at
		FROM vehicle_actions va JOIN users u ON u.id = va.user_id
		WHERE va.vehicle_id = v.id
		ORDER BY va.created_at DESC, va.id DESC
		LIMIT 1
	) a ON true`

const countFrom = `
	FROM vehicles v
	JOIN models m ON m.id = v.model_id
	JOIN makes k ON k.id = m.make_id`

// List implements Store.
func (s *PGStore) List(ctx context.Context, dealershipID string, f Filter, thresholdDays int) ([]Vehicle, int, error) {
	args := []any{dealershipID}
	arg := func(v any) string {
		args = append(args, v)
		return fmt.Sprintf("$%d", len(args))
	}
	// The threshold is bound once, and only when a query actually uses it: Postgres
	// rejects a statement that is given more arguments than it references.
	var thresholdArg string
	threshold := func() string {
		if thresholdArg == "" {
			thresholdArg = arg(thresholdDays) + "::int"
		}
		return thresholdArg
	}

	where := []string{"v.dealership_id = $1::uuid"}
	if f.MakeID != "" {
		where = append(where, "k.id = "+arg(f.MakeID)+"::uuid")
	}
	if f.ModelID != "" {
		where = append(where, "m.id = "+arg(f.ModelID)+"::uuid")
	}
	if f.Aging != nil {
		if *f.Aging {
			where = append(where, "v.stocked_at < CURRENT_DATE - "+threshold())
		} else {
			where = append(where, "v.stocked_at >= CURRENT_DATE - "+threshold())
		}
	}
	if f.MinAgeDays != nil {
		where = append(where, "v.stocked_at <= CURRENT_DATE - "+arg(*f.MinAgeDays)+"::int")
	}
	if f.MaxAgeDays != nil {
		where = append(where, "v.stocked_at >= CURRENT_DATE - "+arg(*f.MaxAgeDays)+"::int")
	}
	clause := " WHERE " + strings.Join(where, " AND ")

	var total int
	if err := s.db.QueryRow(ctx, "SELECT count(*)"+countFrom+clause, args[:len(args):len(args)]...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count vehicles: %w", err)
	}

	order, ok := orderBy[f.Sort]
	if !ok {
		order = orderBy[SortAgeDesc]
	}
	query := fmt.Sprintf(vehicleSelect, threshold()) + clause
	limit, offset := arg(f.PageSize), arg((f.Page-1)*f.PageSize)
	rows, err := s.db.Query(ctx, query+" ORDER BY "+order+", v.id ASC LIMIT "+limit+" OFFSET "+offset, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list vehicles: %w", err)
	}
	defer rows.Close()

	items := []Vehicle{}
	for rows.Next() {
		v, err := scanVehicle(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("list vehicles: %w", err)
	}
	return items, total, nil
}

// Get implements Store.
func (s *PGStore) Get(ctx context.Context, dealershipID, id string, thresholdDays int) (Vehicle, error) {
	v, err := scanVehicle(s.db.QueryRow(ctx,
		fmt.Sprintf(vehicleSelect, "$2::int")+" WHERE v.dealership_id = $1::uuid AND v.id = $3::uuid",
		dealershipID, thresholdDays, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return Vehicle{}, ErrNotFound
	}
	return v, err
}

// Counts implements Store.
func (s *PGStore) Counts(ctx context.Context, dealershipID string, thresholdDays int) (int, int, error) {
	var total, aging int
	err := s.db.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE stocked_at < CURRENT_DATE - $2::int)
		FROM vehicles WHERE dealership_id = $1::uuid`, dealershipID, thresholdDays).Scan(&total, &aging)
	if err != nil {
		return 0, 0, fmt.Errorf("count stock: %w", err)
	}
	return total, aging, nil
}

type scanner interface{ Scan(dest ...any) error }

func scanVehicle(row scanner) (Vehicle, error) {
	var (
		v          Vehicle
		stockedAt  time.Time
		actionID   *string
		actionType *string
		note       *string
		createdBy  *string
		createdAt  *time.Time
	)
	err := row.Scan(&v.ID, &v.VIN, &v.Make.ID, &v.Make.Name, &v.Model.ID, &v.Model.Name, &v.Year, &v.Price,
		&stockedAt, &v.AgeDays, &v.IsAging, &actionID, &actionType, &note, &createdBy, &createdAt)
	if err != nil {
		return Vehicle{}, fmt.Errorf("scan vehicle: %w", err)
	}
	v.StockedAt = stockedAt.Format(time.DateOnly)
	if actionID != nil {
		v.CurrentAction = &actions.Action{
			ID: *actionID, VehicleID: v.ID, ActionType: *actionType, Note: note,
			CreatedBy: *createdBy, CreatedAt: *createdAt,
		}
	}
	return v, nil
}
