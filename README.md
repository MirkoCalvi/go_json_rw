# filereader

A tiny learning-project CLI in Go that persists any JSON value to a file under
`myfiles/` and reads it back. Built to practice the standard `cmd/` + `internal/`
layout and the `encoding/json` package.

## Layout

```
filereader/
├── go.mod                          # module manifest
├── cmd/filereader/main.go          # CLI entry point
├── internal/jsonstore/jsonstore.go # reusable Write/Read helpers
└── myfiles/                        # JSON files live here
```

`internal/` is enforced by the Go compiler — only code inside this module can
import it.

## Build

```bash
go build -o filereader ./cmd/filereader
```

Or run straight from source while iterating:

```bash
go run ./cmd/filereader <args...>
```

## Usage

```
filereader write <filename> [json]   # JSON from stdin OR as the last arg
filereader read  <filename>          # prints pretty-printed JSON to stdout
```

`<filename>` is just the leaf name — files always land in `myfiles/`.

## Examples

```bash
# inline (single-quote the JSON so the shell doesn't split it)
./filereader write alice.json '{"name":"Alice","age":25}'

# from stdin
echo '{"name":"Alice","age":25}' | ./filereader write alice.json

# heredoc for multi-line input
./filereader write notes.json <<'EOF'
{
  "title": "shopping",
  "items": ["bread", "milk"]
}
EOF

# read it back
./filereader read alice.json

# pipe into jq
./filereader read alice.json | jq '.name'
```

## Exit codes

| Code | Meaning |
|------|---------|
| 0    | success |
| 1    | runtime error (bad JSON, missing file, …) |
| 2    | wrong CLI arguments |
