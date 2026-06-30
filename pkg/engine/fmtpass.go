package engine

// FormatPass is a single idempotent byte-level formatting pass. Passes run
// after the core formatter (structural fixes, blank-run collapse,
// trailing-newline normalisation) in registration order; each pass receives
// the output of the previous one.
//
// Implementations must be idempotent: Apply(Apply(src)) == Apply(src).
// src is always valid UTF-8 markdown ending with exactly one trailing newline.
type FormatPass interface {
	// Name returns the unique identifier for this pass, used in diagnostics.
	Name() string
	// Apply transforms src and returns the result.
	Apply(src []byte) []byte
}

// FormatRegistry holds the ordered sequence of FormatPass implementations
// applied after the core formatter. Register passes in the order you want
// them to run; each pass sees the previous pass's output.
type FormatRegistry struct {
	passes []FormatPass
	seen   map[string]bool
}

// NewFormatRegistry returns an empty registry.
func NewFormatRegistry() *FormatRegistry {
	return &FormatRegistry{seen: map[string]bool{}}
}

// Register appends a pass. Panics on duplicate names (programmer error).
func (r *FormatRegistry) Register(p FormatPass) {
	name := p.Name()
	if r.seen[name] {
		panic("duplicate format pass: " + name)
	}
	r.seen[name] = true
	r.passes = append(r.passes, p)
}

// All returns the registered passes in registration order.
func (r *FormatRegistry) All() []FormatPass { return r.passes }

// DefaultFormatRegistry returns a FormatRegistry pre-loaded with the
// built-in passes in their canonical order:
//
//  1. table-align      — aligns GFM table column widths
//  2. shortcode-indent — re-indents Hugo shortcode tag lines by nesting depth
//
// To add a new built-in pass, implement FormatPass in its own file and add a
// Register call here. To use a custom set, start with NewFormatRegistry.
func DefaultFormatRegistry() *FormatRegistry {
	reg := NewFormatRegistry()
	reg.Register(TableAlignPass{})
	reg.Register(ShortcodeIndentPass{})
	return reg
}
