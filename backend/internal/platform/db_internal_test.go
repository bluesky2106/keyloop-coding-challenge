package platform

import "testing"

func TestToPgx5(t *testing.T) {
	tests := map[string]struct {
		want string
		ok   bool
	}{
		"postgres://u:p@h:5432/db?sslmode=disable": {"pgx5://u:p@h:5432/db?sslmode=disable", true},
		"postgresql://u:p@h/db":                    {"pgx5://u:p@h/db", true},
		"host=localhost user=u dbname=db":          {"", false},
		"mysql://u@h/db":                           {"", false},
	}
	for in, tt := range tests {
		got, ok := toPgx5(in)
		if got != tt.want || ok != tt.ok {
			t.Errorf("toPgx5(%q) = %q, %v; want %q, %v", in, got, ok, tt.want, tt.ok)
		}
	}
}
