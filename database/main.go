package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/PhantomX7/athleton/internal/models"

	"ariga.io/atlas-provider-gorm/gormschema"
)

// atlas migrate diff --env gorm
// for auto creating migration file
func main() {
	// First, output the static schema
	staticSchemaPath := "database/schema/schema.sql"
	if content, err := os.ReadFile(staticSchemaPath); err == nil {
		fmt.Println(string(content))
		fmt.Println("\n-- GORM Generated Schema Below --")
	} else if !os.IsNotExist(err) {
		log.Fatalf("failed to open static schema: %v", err)
	}

	stmts, err := gormschema.New("postgres").Load(
		&models.User{},
		&models.RefreshToken{},
		&models.Config{},
		&models.Log{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	if _, err := io.WriteString(os.Stdout, stmts); err != nil {
		log.Fatalf("failed to write schema to stdout: %v", err)
	}
}
