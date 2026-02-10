// Package web provides embedded frontend assets for Life Pilot.
// Build the frontend with `bun run build` before building the Go binary.
package web

import "embed"

//go:embed dist/*
var Assets embed.FS
