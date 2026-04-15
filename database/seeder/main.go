package main

import (
	"log"

	"github.com/PhantomX7/athleton/database/seeder/seed"
	"github.com/PhantomX7/athleton/internal/bootstrap"
	"github.com/PhantomX7/athleton/pkg/config"

	"gorm.io/gorm"
)

const BcryptCost = 12

func main() {
	log.Println("Starting seeder...")

	// Set up config
	if err := bootstrap.SetUpConfig(); err != nil {
		log.Fatalf("Failed to set up config: %v", err)
	}

	// Set up logger immediately after config (all other setup functions need logger)
	if err := bootstrap.SetUpLogger(); err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}

	db, err := bootstrap.SetUpDatabase(nil)
	if err != nil {
		log.Fatalf("Failed to set up database: %v", err)
	}

	// Seed data
	if err := seedData(db); err != nil {
		log.Fatalf("Failed to seed data: %v", err)
	}

	log.Println("Seeding completed successfully!")
}

func seedData(db *gorm.DB) error {
	log.Println("Seeding users...")
	if err := seed.SeedUsers(db); err != nil {
		return err
	}

	log.Println("Seeding configs...")
	if err := seed.SeedConfigs(db); err != nil {
		return err
	}

	if config.Get().IsDevelopment() {
		log.Println("Seeding brands...")

	}

	return nil
}
