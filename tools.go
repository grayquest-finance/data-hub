//go:build tools

package tools

// This file pins tool dependencies so their full dependency tree is
// included in go.sum, which is required for `go run` in Go 1.21+.
import _ "github.com/99designs/gqlgen"