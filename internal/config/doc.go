// Package config loads, validates, and atomically reloads application settings.
//
// A Manager always exposes one immutable Settings snapshot. A candidate TOML
// file replaces that snapshot only after the whole file validates and a
// protected last-valid copy has been written successfully.
package config
