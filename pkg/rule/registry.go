package rule

// Registry holds rules by name in registration order.
type Registry struct {
	rules map[string]Rule
	order []string
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{rules: map[string]Rule{}}
}

// Register adds a rule; a duplicate name panics (programmer error).
func (r *Registry) Register(rule Rule) {
	name := rule.Meta().Name
	if _, ok := r.rules[name]; ok {
		panic("duplicate rule registration: " + name)
	}
	r.rules[name] = rule
	r.order = append(r.order, name)
}

// Get returns the rule with name, if present.
func (r *Registry) Get(name string) (Rule, bool) {
	rule, ok := r.rules[name]
	return rule, ok
}

// All returns the rules in registration order.
func (r *Registry) All() []Rule {
	out := make([]Rule, 0, len(r.order))
	for _, n := range r.order {
		out = append(out, r.rules[n])
	}
	return out
}
