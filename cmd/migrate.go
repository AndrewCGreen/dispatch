package cmd

import (
	"fmt"
	"os"

	"github.com/dispatch-email/dispatch/internal/config"
	"github.com/dispatch-email/dispatch/internal/store"
)

func runMigrate() error {
	cfgPath := "dispatch.yaml"
	if len(os.Args) > 2 {
		cfgPath = os.Args[2]
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	db, err := store.New(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Println("✓ Migrations complete")
	return nil
}
