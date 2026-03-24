package internal

import "fmt"

// Router maps Model aliases to their LlmClient implementations.
type Router struct {
	clients map[Model]LlmClient
}

// NewRouter creates an empty Router.
func NewRouter() *Router {
	return &Router{clients: make(map[Model]LlmClient)}
}

// Register associates a model alias with a client.
func (r *Router) Register(model Model, client LlmClient) {
	r.clients[model] = client
}

// Get returns the LlmClient for the given model alias.
func (r *Router) Get(model Model) (LlmClient, error) {
	c, ok := r.clients[model]
	if !ok {
		return nil, fmt.Errorf("no client registered for model %q", model)
	}
	return c, nil
}
