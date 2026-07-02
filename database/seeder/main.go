// Package main runs the database seeder entrypoint.
package main

import (
	"log"

	"github.com/PhantomX7/athleton/database/seeder/seed"
	"github.com/PhantomX7/athleton/internal/bootstrap"
	"github.com/PhantomX7/athleton/pkg/config"

	"gorm.io/gorm"
)

func main() {
	log.Println("Starting seeder...")

	// Load config first (logger and database need it).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Set up logger immediately after config (all other setup functions need logger)
	if err := bootstrap.SetUpLogger(cfg); err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}

	db, err := bootstrap.SetUpDatabase(nil, cfg)
	if err != nil {
		log.Fatalf("Failed to set up database: %v", err)
	}

	// Seed data
	if err := seedData(db, cfg); err != nil {
		log.Fatalf("Failed to seed data: %v", err)
	}

	log.Println("Seeding completed successfully!")
}

func seedData(db *gorm.DB, cfg *config.Config) error {
	log.Println("Seeding users...")
	if err := seed.SeedUsers(db, cfg); err != nil {
		return err
	}

	log.Println("Seeding configs...")
	if err := seed.SeedConfigs(db); err != nil {
		return err
	}

	if cfg.IsDevelopment() {
		log.Println("Seeding brands...")

	}

	return nil
}
