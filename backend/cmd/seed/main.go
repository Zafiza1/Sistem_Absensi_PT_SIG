// Command seed creates (or updates) the initial SUPER_ADMIN account so the
// dashboard has someone who can log in on a fresh database. It is
// idempotent: running it again just resets that account's password.
//
// Usage:
//
//	SEED_ADMIN_EMAIL=admin@suryaintigas.com SEED_ADMIN_PASSWORD='...' go run ./cmd/seed
//
// SEED_ADMIN_* values are read directly from the environment, never
// hardcoded, so no real credential is ever committed to source control. In
// non-production environments, omitted values fall back to clearly-logged
// development defaults; production requires them to be set explicitly.
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"github.com/suryaintigas/absensi-backend/internal/config"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	name := os.Getenv("SEED_ADMIN_NAME")
	email := os.Getenv("SEED_ADMIN_EMAIL")
	password := os.Getenv("SEED_ADMIN_PASSWORD")

	if cfg.IsProduction() && (email == "" || password == "") {
		log.Fatal("SEED_ADMIN_EMAIL and SEED_ADMIN_PASSWORD are required when APP_ENV=production")
	}
	if name == "" {
		name = "Super Admin"
	}
	if email == "" {
		email = "admin@suryaintigas.local"
		log.Printf("SEED_ADMIN_EMAIL not set, using development default: %s", email)
	}
	if password == "" {
		password = "ChangeMe123!"
		log.Print("SEED_ADMIN_PASSWORD not set, using development default (change it after first login)")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("failed to hash password: %v", err)
	}

	const q = `
		INSERT INTO users (name, email, password_hash, role, is_active)
		VALUES ($1, $2, $3, 'SUPER_ADMIN', true)
		ON CONFLICT (email) DO UPDATE
			SET password_hash = EXCLUDED.password_hash,
			    role = 'SUPER_ADMIN',
			    is_active = true,
			    updated_at = now()`

	if _, err := pool.Exec(ctx, q, name, email, string(hash)); err != nil {
		log.Fatalf("failed to seed super admin: %v", err)
	}

	log.Printf("seeded SUPER_ADMIN account: %s", email)
}
