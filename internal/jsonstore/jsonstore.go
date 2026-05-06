// Package jsonstore provides helpers to persist any Go value as JSON on disk
// and to read it back into a Go value. It lives under "internal/" so only code
// inside this module ("filereader/...") is allowed to import it — Go enforces
// this rule at compile time, which is the standard way to mark a package as
// "implementation detail, not public API".
package jsonstore

// Standard library imports we need:
//   - encoding/json : marshalling Go values <-> JSON bytes
//   - fmt           : building wrapped error messages
//   - os            : file system operations (MkdirAll, WriteFile, ReadFile)
//   - path/filepath : OS-agnostic path joining ("a/b" on Linux, "a\b" on Windows)
import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BaseDir is the directory (relative to the working directory of the running
// program) where every JSON file produced by this package will live. Exposing
// it as a package-level variable makes it easy to override in tests.
const BaseDir = "myfiles"

// Write serialises any Go value `data` to JSON and saves it to
// "<BaseDir>/<filename>". The `data` parameter has type `any` (alias for
// `interface{}`), which means callers can hand us literally any struct, map,
// slice, etc. — encoding/json uses reflection to figure out the shape.
func Write(filename string, data any) error {
	// Make sure the destination directory exists. MkdirAll is idempotent: it
	// does nothing if the dir already exists, and creates parent directories
	// as needed. 0o755 = owner can read/write/list, others can read/list.
	if err := os.MkdirAll(BaseDir, 0o755); err != nil {
		// fmt.Errorf with %w wraps the underlying error so callers can still
		// inspect it with errors.Is / errors.As if they need to.
		return fmt.Errorf("creating %q: %w", BaseDir, err)
	}

	// Convert the Go value into JSON bytes. MarshalIndent produces a
	// human-readable, pretty-printed JSON document (prefix="", indent="  ").
	// Use json.Marshal instead if you want the compact single-line form.
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}

	// Build the full path in an OS-agnostic way: filepath.Join uses the right
	// separator for the current platform.
	fullPath := filepath.Join(BaseDir, filename)

	// WriteFile creates (or truncates) the file and writes all bytes in one
	// call — no need to manually Open/Close. 0o644 = owner read/write,
	// everyone else read-only, which is the conventional permission for
	// regular data files.
	if err := os.WriteFile(fullPath, bytes, 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", fullPath, err)
	}

	// nil error = success in Go's idiomatic error-handling style.
	return nil
}

// Read loads the JSON document at "<BaseDir>/<filename>" and decodes it into
// whatever Go value `out` points to. `out` MUST be a pointer (e.g. &myStruct)
// because json.Unmarshal needs to mutate the destination — passing a value
// directly would only modify a local copy.
func Read(filename string, out any) error {
	// Build the full path the same way Write did, so the two operations stay
	// symmetrical.
	fullPath := filepath.Join(BaseDir, filename)

	// ReadFile slurps the whole file into memory. Fine for small config-like
	// files; for very large files you'd stream with json.NewDecoder instead.
	bytes, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("reading %q: %w", fullPath, err)
	}

	// Decode the JSON into the caller's destination. Unmarshal uses reflection
	// on `out` to figure out which fields to populate; unknown JSON fields are
	// silently ignored, missing fields keep their zero value.
	if err := json.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("decoding JSON from %q: %w", fullPath, err)
	}

	return nil
}
