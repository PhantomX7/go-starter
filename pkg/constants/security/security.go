// Package security centralizes security-related tuning constants shared
// across modules, so values like the bcrypt work factor cannot drift between
// the flows that hash passwords.
package security

// BcryptCost is the bcrypt work factor used for every password hash in the
// application (registration, self-service password changes, and admin-driven
// password resets). Keep all hashing on the same cost so every stored hash
// ages together and can be rotated with a single change here.
const BcryptCost = 12
