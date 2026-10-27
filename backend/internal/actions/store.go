// Package actions records and lists the actions managers log against aging vehicles.
package actions

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound means the vehicle does not exist or belongs to another dealership.
var ErrNotFound = errors.New("vehicle not found")

// ErrNotAging means the vehicle is not older than the aging threshold, so it does not
// accept actions.
var ErrNotAging = errors.New("vehicle is not aging")

// Action is one entry in a vehicle's append-only history.
type Action struct {
	ID         string    `json:"id"`
	VehicleID  string    `json:"vehicle_id"`
	ActionType string    `json:"action_type"`
	Note       *string   `json:"note"`
	CreatedBy  string    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

// NewAction is what a caller supplies to log an action.
type NewAction struct {
	DealershipID string
	UserID       string
	VehicleID    string
	ActionType   string
	Note         *string
}

// Store reads and appends actions. Every method is scoped to a dealership.
type Store interface {
	// List returns the vehicle's history, newest first, or ErrNotFound.
	List(ctx context.Context, dealershipID, vehicleID string) ([]Action, error)
	// Create appends an action to an aging vehicle. It returns ErrNotFound for a
	// vehicle outside the dealership and ErrNotAging for a vehicle that is not older
	// than thresholdDays.
	Create(ctx context.Context, a NewAction, thresholdDays int) (Action, error)
}

// PGStore is the PostgreSQL implementation of Store.
type PGStore struct{ db *pgxpool.Pool }

// NewPGStore returns a Store backed by db.
func NewPGStore(db *pgxpool.Pool) *PGStore { return &PGStore{db: db} }

// List implements Store.
func (s *PGStore) List(ctx context.Context, dealershipID, vehicleID string) ([]Action, error) {
	var exists bool
	if err := s.db.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM vehicles WHERE id = $1::uuid AND dealership_id = $2::uuid)`,
		vehicleID, dealershipID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("check vehicle: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}

	rows, err := s.db.Query(ctx, `
		SELECT a.id::text, a.vehicle_id::text, a.action_type, a.note, u.username, a.created_at
		FROM vehicle_actions a JOIN users u ON u.id = a.user_id
		WHERE a.vehicle_id = $1::uuid AND a.dealership_id = $2::uuid
		ORDER BY a.created_at DESC, a.id DESC`, vehicleID, dealershipID)
	if err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}
	defer rows.Close()

	out := []Action{}
	for rows.Next() {
		var a Action
		if err := rows.Scan(&a.ID, &a.VehicleID, &a.ActionType, &a.Note, &a.CreatedBy, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan action: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list actions: %w", err)
	}
	return out, nil
}

// Create implements Store.
func (s *PGStore) Create(ctx context.Context, n NewAction, thresholdDays int) (Action, error) {
	var aging bool
	err := s.db.QueryRow(ctx, `
		SELECT stocked_at < CURRENT_DATE - $3::int
		FROM vehicles WHERE id = $1::uuid AND dealership_id = $2::uuid`,
		n.VehicleID, n.DealershipID, thresholdDays).Scan(&aging)
	if errors.Is(err, pgx.ErrNoRows) {
		return Action{}, ErrNotFound
	}
	if err != nil {
		return Action{}, fmt.Errorf("check vehicle: %w", err)
	}
	if !aging {
		return Action{}, ErrNotAging
	}

	var a Action
	err = s.db.QueryRow(ctx, `
		WITH ins AS (
			INSERT INTO vehicle_actions (vehicle_id, dealership_id, user_id, action_type, note)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5)
			RETURNING id, vehicle_id, user_id, action_type, note, created_at
		)
		SELECT ins.id::text, ins.vehicle_id::text, ins.action_type, ins.note, u.username, ins.created_at
		FROM ins JOIN users u ON u.id = ins.user_id`,
		n.VehicleID, n.DealershipID, n.UserID, n.ActionType, n.Note).
		Scan(&a.ID, &a.VehicleID, &a.ActionType, &a.Note, &a.CreatedBy, &a.CreatedAt)
	if err != nil {
		return Action{}, fmt.Errorf("insert action: %w", err)
	}
	return a, nil
}
