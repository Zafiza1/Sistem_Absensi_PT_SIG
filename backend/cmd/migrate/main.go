// Command migrate applies or rolls back database migrations without
// requiring the separate golang-migrate CLI binary to be installed — it
// links the same library the server itself uses.
//
// Usage:
//
//	go run ./cmd/migrate -direction up
//	go run ./cmd/migrate -direction down
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"

	"github.com/suryaintigas/absensi-backend/internal/config"
	"github.com/suryaintigas/absensi-backend/internal/database"
)

func main() {
	direction := flag.String("direction", "up", `migration direction: "up" or "down" (down rolls back one step)`)
	flag.Parse()

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	migrationsPath, err := filepath.Abs("migrations")
	if err != nil {
		log.Fatalf("could not resolve migrations path: %v", err)
	}

	switch *direction {
	case "up":
		if err := database.MigrateUp(cfg.DatabaseURL, migrationsPath); err != nil {
			log.Fatalf("migrate up failed: %v", err)
		}
		log.Println("migrations applied successfully")
	case "down":
		if err := database.MigrateDown(cfg.DatabaseURL, migrationsPath); err != nil {
			log.Fatalf("migrate down failed: %v", err)
		}
		log.Println("rolled back one migration")
	default:
		log.Fatalf("unknown -direction %q (expected \"up\" or \"down\")", *direction)
		os.Exit(2)
	}
}
