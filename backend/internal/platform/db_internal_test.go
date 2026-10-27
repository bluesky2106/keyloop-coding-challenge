package platform

import "testing"

func TestToPgx5(t *testing.T) {
	tests := map[string]string{
		"postgres://u:p@h:5432/db?sslmode=disable":    "pgx5://u:p@h:5432/db?sslmode=disable",
		"postgresql://u:p@h/db":                       "pgx5://u:p@h/db",
		"pgx5://already/db":                           "pgx5://already/db",
		"host=localhost user=u dbname=db (key/value)": "host=localhost user=u dbname=db (key/value)",
	}
	for in, want := range tests {
		if got := toPgx5(in); got != want {
			t.Errorf("toPgx5(%q) = %q, want %q", in, got, want)
		}
	}
}
