// Package integration contains end-to-end integration tests that exercise the
// assembled application — the real Gin engine built by bootstrap.SetupServer,
// the full middleware chain, and real services/repositories — against an
// in-memory SQLite database. All behavior lives in the *_test.go files.
package integration
