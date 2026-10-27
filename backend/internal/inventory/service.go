package inventory

import "context"

// Page is one page of a filtered vehicle list.
type Page struct {
	Items    []Vehicle `json:"items"`
	Page     int       `json:"page"`
	PageSize int       `json:"page_size"`
	Total    int       `json:"total"`
}

// Summary is the dealership's unfiltered stock totals.
type Summary struct {
	Total              int `json:"total"`
	Aging              int `json:"aging"`
	AgingThresholdDays int `json:"aging_threshold_days"`
}

// Service applies the aging threshold to the store.
type Service struct {
	store         Store
	thresholdDays int
}

// NewService returns a Service. A vehicle is aging when it is older than thresholdDays.
func NewService(store Store, thresholdDays int) *Service {
	return &Service{store: store, thresholdDays: thresholdDays}
}

// List returns one page of the dealership's vehicles.
func (s *Service) List(ctx context.Context, dealershipID string, f Filter) (Page, error) {
	items, total, err := s.store.List(ctx, dealershipID, f, s.thresholdDays)
	if err != nil {
		return Page{}, err
	}
	return Page{Items: items, Page: f.Page, PageSize: f.PageSize, Total: total}, nil
}

// Get returns one vehicle, or ErrNotFound.
func (s *Service) Get(ctx context.Context, dealershipID, id string) (Vehicle, error) {
	return s.store.Get(ctx, dealershipID, id, s.thresholdDays)
}

// Summary returns the dealership's stock and aging totals.
func (s *Service) Summary(ctx context.Context, dealershipID string) (Summary, error) {
	total, aging, err := s.store.Counts(ctx, dealershipID, s.thresholdDays)
	if err != nil {
		return Summary{}, err
	}
	return Summary{Total: total, Aging: aging, AgingThresholdDays: s.thresholdDays}, nil
}
