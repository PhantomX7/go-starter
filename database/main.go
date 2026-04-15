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
	if file, err := os.Open(staticSchemaPath); err == nil {
		defer file.Close()
		content, err := io.ReadAll(file)
		if err != nil {
			log.Fatalf("failed to read static schema: %v", err)
		}
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
	io.WriteString(os.Stdout, stmts)
}
