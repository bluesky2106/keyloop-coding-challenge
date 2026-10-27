package seed

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"github.com/bluesky2106/keyloop-coding-challenge/backend/internal/testdb"
)

const seedDir = "../../seed"

func TestLoadIsIdempotentAndHashesPasswords(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()

	counts := func() [5]int {
		var c [5]int
		for i, q := range []string{
			"SELECT count(*) FROM dealerships", "SELECT count(*) FROM users",
			"SELECT count(*) FROM makes", "SELECT count(*) FROM models", "SELECT count(*) FROM vehicles",
		} {
			if err := pool.QueryRow(ctx, q).Scan(&c[i]); err != nil {
				t.Fatalf("%s: %v", q, err)
			}
		}
		return c
	}

	if err := Load(ctx, pool, seedDir); err != nil {
		t.Fatalf("first load: %v", err)
	}
	first := counts()
	if first != [5]int{3, 6, 7, 21, 150} {
		t.Fatalf("counts after first load = %v", first)
	}
	if err := Load(ctx, pool, seedDir); err != nil {
		t.Fatalf("second load: %v", err)
	}
	if second := counts(); second != first {
		t.Fatalf("counts changed on reload: %v -> %v", first, second)
	}

	var hash string
	if err := pool.QueryRow(ctx,
		`SELECT password_hash FROM users WHERE username = 'northside.manager'`).Scan(&hash); err != nil {
		t.Fatal(err)
	}
	if hash == "demo-northside-1" || bcrypt.CompareHashAndPassword([]byte(hash), []byte("demo-northside-1")) != nil {
		t.Fatal("password is not stored as a bcrypt hash of the CSV value")
	}

	// Boundary vehicles: 90 days is not aging, 91 is.
	var aging90, aging91 bool
	if err := pool.QueryRow(ctx, `
		SELECT bool_or(CURRENT_DATE - stocked_at = 90 AND stocked_at < CURRENT_DATE - 90),
		       bool_or(CURRENT_DATE - stocked_at = 91 AND stocked_at < CURRENT_DATE - 90)
		FROM vehicles`).Scan(&aging90, &aging91); err != nil {
		t.Fatal(err)
	}
	if aging90 || !aging91 {
		t.Fatalf("aging boundary wrong: age 90 aging=%v, age 91 aging=%v", aging90, aging91)
	}
}

func TestLoadRejectsUnknownReferences(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("dealerships.csv", "name\nA\n")
	write("models.csv", "make,model\nToyota,Corolla\n")
	write("users.csv", "dealership,username,password\nA,u,p\n")
	write("vehicles.csv", "dealership,vin,make,model,year,price,age_days\n"+
		"A,11111111111111111,Toyota,Missing,2024,1.00,5\n")

	err := Load(context.Background(), nil, dir)
	if err == nil || !strings.Contains(err.Error(), "unknown model") {
		t.Fatalf("err = %v, want unknown model", err)
	}
}
