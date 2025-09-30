package main

import (
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"
	"github.com/PhantomX7/go-starter/internal/models"
)

// atlas migrate diff --env gorm
// for auto creating migration file
func main() {
	stmts, err := gormschema.New("postgres").Load(&models.Post{})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}
	io.WriteString(os.Stdout, stmts)
}
