//go:build wasm && !persist

package api

// StringArray degrades to a plain []string for the bare-wasm (browser) build,
// which cannot import github.com/lib/pq. The JSON/XML wire shape is identical
// to pq.StringArray (whose underlying type is []string); Postgres array
// encoding is N/A in this build (no DB). See delivery_persist.go.
type StringArray = []string
