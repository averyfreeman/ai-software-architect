// Package templates contains the embedded project-scaffold assets.
package templates

import "embed"

// FS contains the catalog and all scaffold templates.
//
//go:embed catalog.yaml */*
var FS embed.FS
