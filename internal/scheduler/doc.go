// Package scheduler coordinates minute-aligned host and Docker collection.
// It prevents overlapping runs, applies deadlines and cancellation, and keeps
// successful subsystem results when another subsystem fails.
package scheduler
