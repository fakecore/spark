package llm

import "fmt"

// Registry is a minimal provider registry to decouple domain logic from concrete implementations.
type Registry struct {
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: map[string]Provider{}}
}

func (r *Registry) Register(name string, p Provider) error {
	if name == "" {
		return fmt.Errorf("provider name is required")
	}
	if p == nil {
		return fmt.Errorf("provider is nil")
	}
	if _, ok := r.providers[name]; ok {
		return fmt.Errorf("provider already registered: %q", name)
	}
	r.providers[name] = p
	return nil
}

func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.providers[name]
	return p, ok
}
