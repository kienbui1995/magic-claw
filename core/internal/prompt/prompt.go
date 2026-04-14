// Package prompt provides a versioned prompt template registry with
// variable interpolation and A/B testing support.
package prompt

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// Template is a versioned prompt template.
type Template struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Version   int               `json:"version"`
	Content   string            `json:"content"`   // template with {{variable}} placeholders
	Metadata  map[string]string `json:"metadata,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
}

// ABTest defines an A/B test between prompt variants.
type ABTest struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Variants []string `json:"variants"` // template IDs
	Weights  []int    `json:"weights"`  // relative weights (e.g., [80, 20])
	Active   bool     `json:"active"`
}

// Registry stores and retrieves prompt templates.
type Registry struct {
	mu        sync.RWMutex
	templates map[string]*Template // id -> template
	byName    map[string][]string  // name -> [id_v1, id_v2, ...]
	tests     map[string]*ABTest   // test id -> test
}

// NewRegistry creates a new prompt registry.
func NewRegistry() *Registry {
	return &Registry{
		templates: make(map[string]*Template),
		byName:    make(map[string][]string),
		tests:     make(map[string]*ABTest),
	}
}

// Add stores a new prompt template. Auto-increments version for same name.
func (r *Registry) Add(name, content string, metadata map[string]string) *Template {
	r.mu.Lock()
	defer r.mu.Unlock()

	version := len(r.byName[name]) + 1
	id := fmt.Sprintf("%s:v%d", name, version)

	t := &Template{
		ID:        id,
		Name:      name,
		Version:   version,
		Content:   content,
		Metadata:  metadata,
		CreatedAt: time.Now().UTC(),
	}
	r.templates[id] = t
	r.byName[name] = append(r.byName[name], id)
	return t
}

// Get returns a template by ID.
func (r *Registry) Get(id string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.templates[id]
	if !ok {
		return nil, fmt.Errorf("template %q not found", id)
	}
	return t, nil
}

// Latest returns the latest version of a named template.
func (r *Registry) Latest(name string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ids, ok := r.byName[name]
	if !ok || len(ids) == 0 {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return r.templates[ids[len(ids)-1]], nil
}

// List returns all templates.
func (r *Registry) List() []*Template {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*Template, 0, len(r.templates))
	for _, t := range r.templates {
		result = append(result, t)
	}
	return result
}

// Render interpolates variables into a template.
// Variables use {{name}} syntax.
func Render(tmpl string, vars map[string]string) string {
	result := tmpl
	for k, v := range vars {
		result = strings.ReplaceAll(result, "{{"+k+"}}", v)
	}
	return result
}

// AddABTest creates an A/B test between template variants.
func (r *Registry) AddABTest(id, name string, variants []string, weights []int) *ABTest {
	r.mu.Lock()
	defer r.mu.Unlock()
	test := &ABTest{ID: id, Name: name, Variants: variants, Weights: weights, Active: true}
	r.tests[id] = test
	return test
}

// Resolve picks a template — if an active A/B test exists for the name,
// selects a variant by weight; otherwise returns latest version.
func (r *Registry) Resolve(name string) (*Template, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check A/B tests
	for _, test := range r.tests {
		if test.Name == name && test.Active && len(test.Variants) > 0 {
			id := weightedPick(test.Variants, test.Weights)
			if t, ok := r.templates[id]; ok {
				return t, nil
			}
		}
	}

	// Fall back to latest
	ids, ok := r.byName[name]
	if !ok || len(ids) == 0 {
		return nil, fmt.Errorf("template %q not found", name)
	}
	return r.templates[ids[len(ids)-1]], nil
}

func weightedPick(items []string, weights []int) string {
	if len(weights) != len(items) {
		return items[rand.Intn(len(items))]
	}
	total := 0
	for _, w := range weights {
		total += w
	}
	r := rand.Intn(total)
	for i, w := range weights {
		r -= w
		if r < 0 {
			return items[i]
		}
	}
	return items[len(items)-1]
}
