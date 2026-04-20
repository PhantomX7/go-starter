// This file owns the GORM CLI field-helper generation for the models package.
// Running `go generate ./internal/models/...` (or `make gorm-gen`) reads every
// model struct in this directory and writes field-helper code into
// ../generated. Do not edit the output by hand.
//
// If you ever need SQL-template interfaces as well, create a sibling package
// (e.g. internal/queries), drop its own //go:generate directive there, and
// point -o at the same generated dir.

package models

//go:generate gorm gen -i . -o ../generated
