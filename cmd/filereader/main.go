// Package main is the CLI entry point. Usage:
//
//	filereader write <filename> [json]    # JSON from stdin OR as a final arg
//	filereader read  <filename>           # loads myfiles/<filename>, prints to stdout
//
// Examples:
//
//	echo '{"name":"Alice","age":25}' | filereader write alice.json
//	filereader write alice.json '{"name":"Alice","age":25}'
//	filereader read alice.json
package main

// Standard-library imports:
//   - encoding/json : pretty-print the map we read back
//   - fmt           : write to stdout / stderr
//   - io            : io.ReadAll to slurp the whole stdin
//   - os            : os.Args (CLI arguments), os.Stdin, os.Exit
//   - strings       : strings.NewReader, to wrap an inline JSON arg as an io.Reader
//
// Plus our own package, reached via the module path from go.mod.
import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"filereader/internal/jsonstore"
)

// usage prints how the tool should be invoked and exits with a non-zero
// status. Conventionally, help/error messages go to stderr (fd 2), not stdout,
// so they don't pollute pipelines like `filereader read x.json | jq ...`.
func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  filereader write <filename> [json]   # JSON on stdin OR as last arg")
	fmt.Fprintln(os.Stderr, "  filereader read  <filename>")
	os.Exit(2) // exit code 2 is the typical "bad CLI usage" signal
}

func main() {
	// os.Args[0] is the program name; user arguments start at index 1. We
	// need at least <subcommand> <filename> (so len >= 3). The exact upper
	// bound depends on the subcommand and is checked below.
	if len(os.Args) < 3 {
		usage()
	}

	// Pull out the common arguments. Naming them locally makes the dispatch
	// switch below much easier to read.
	subcommand := os.Args[1]
	filename := os.Args[2]

	// Dispatch on the subcommand. Go's switch doesn't fall through by default,
	// so each case is self-contained — no `break` needed.
	switch subcommand {
	case "write":
		// Decide where the JSON body comes from:
		//   - 3 args (program write file)         → read from stdin (pipe / heredoc)
		//   - 4 args (program write file json)    → use the 4th arg as the body
		// `src` is typed as io.Reader so runWrite doesn't care which case we
		// took; strings.NewReader wraps a string into an io.Reader for free.
		var src io.Reader
		switch len(os.Args) {
		case 3:
			src = os.Stdin
		case 4:
			src = strings.NewReader(os.Args[3])
		default:
			usage() // too many args
		}

		if err := runWrite(filename, src); err != nil {
			// fmt.Fprintf to stderr + exit 1 is the conventional way to fail
			// a CLI command from main.
			fmt.Fprintf(os.Stderr, "write: %v\n", err)
			os.Exit(1)
		}

	case "read":
		// `read` doesn't take a body, so any extra argument is a usage error.
		if len(os.Args) != 3 {
			usage()
		}
		// runRead writes its output to whatever io.Writer we pass — using
		// os.Stdout here means the result can be piped into other tools.
		if err := runRead(filename, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "read: %v\n", err)
			os.Exit(1)
		}

	default:
		// Anything other than "write" or "read" is a usage error.
		usage()
	}
}

// runWrite reads a JSON document from `in` and saves it to myfiles/<filename>.
// It accepts any JSON value at the top level (object, array, …) by decoding
// into `any`, so the CLI is not tied to a specific struct schema.
func runWrite(filename string, in io.Reader) error {
	// io.ReadAll consumes the entire reader until EOF and returns the bytes.
	// For stdin in a CLI this is fine: the user closes stdin (Ctrl-D, end of
	// pipe) when they're done.
	raw, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	// Decode into `any` purely to *validate* that the input is well-formed
	// JSON before we save it. If we wrote `raw` to disk directly we'd happily
	// persist garbage like "not json at all".
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("input is not valid JSON: %w", err)
	}

	// Delegate to the jsonstore package. It re-encodes (pretty-printed) and
	// handles directory creation, so the CLI doesn't have to know about paths
	// or file permissions.
	if err := jsonstore.Write(filename, payload); err != nil {
		return err
	}

	// Friendly confirmation on stderr — keeps stdout clean for tools that
	// might pipe the write subcommand somewhere.
	fmt.Fprintf(os.Stderr, "wrote %s/%s\n", jsonstore.BaseDir, filename)
	return nil
}

// runRead loads myfiles/<filename> and writes a pretty-printed JSON copy to
// `out`. We decode into `any` so the function works for any shape of JSON.
func runRead(filename string, out io.Writer) error {
	// Destination for the decoded JSON. `any` (alias for interface{}) is the
	// catch-all type the encoding/json package uses for "I don't know the
	// schema in advance".
	var payload any
	if err := jsonstore.Read(filename, &payload); err != nil {
		return err
	}

	// Re-encode for display. MarshalIndent gives a nicely formatted output
	// instead of one long line. We could also have stored the raw bytes and
	// streamed them out unchanged — going through Marshal proves the file
	// round-trips cleanly through Go types.
	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding output: %w", err)
	}

	// Write the JSON followed by a newline so terminals show a clean prompt
	// after the output. Fprintln adds the '\n' for us.
	_, err = fmt.Fprintln(out, string(bytes))
	return err
}
