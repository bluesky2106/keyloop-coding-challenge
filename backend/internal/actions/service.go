package actions

import "context"

// Service applies the aging rule to the store.
type Service struct {
	store         Store
	thresholdDays int
}

// NewService returns a Service. thresholdDays is the aging threshold: a vehicle accepts
// actions only when it is older than this many days.
func NewService(store Store, thresholdDays int) *Service {
	return &Service{store: store, thresholdDays: thresholdDays}
}

// List returns the vehicle's history, newest first.
func (s *Service) List(ctx context.Context, dealershipID, vehicleID string) ([]Action, error) {
	return s.store.List(ctx, dealershipID, vehicleID)
}

// Create logs an action against an aging vehicle.
func (s *Service) Create(ctx context.Context, n NewAction) (Action, error) {
	return s.store.Create(ctx, n, s.thresholdDays)
}
