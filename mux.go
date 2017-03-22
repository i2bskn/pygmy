package tensile

import (
	"net/http"
	"path"
	"sync"
)

// Mux is http.ServeMux compatible HTTP multiplexer.
type Mux struct {
	mu         sync.RWMutex
	m          *node
	middleware []func(http.Handler) http.Handler
}

// New allocates and returns a new Mux.
func New() *Mux {
	return new(Mux)
}

// Handle registers handler and returns Entry.
func (mux *Mux) Handle(pattern string, h http.Handler) *Entry {
	mux.mu.Lock()
	defer mux.mu.Unlock()

	if pattern == "" {
		panic("http: invalid pattern " + pattern)
	}

	if h == nil {
		panic("http: nil handler")
	}

	if mux.m == nil {
		mux.m = new(node)
	}

	p := cleanPath(pattern)
	e := newEntry(p, h)
	mux.m.add(p, e)
	return e
}

// HandleFunc registers handler function and returns Entry.
func (mux *Mux) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) *Entry {
	return mux.Handle(pattern, http.HandlerFunc(h))
}

// ServeHTTP dispatches matching requests to handlers.
func (mux *Mux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h, r := mux.Handler(r)
	for i := len(mux.middleware) - 1; i >= 0; i-- {
		h = mux.middleware[i](h)
	}
	h.ServeHTTP(w, r)
}

// Handler returns a handler to dispatch from request.
func (mux *Mux) Handler(r *http.Request) (http.Handler, *http.Request) {
	mux.mu.RLock()
	defer mux.mu.RUnlock()

	if e, r := mux.m.match(r.URL.Path, r); e != nil {
		return e.h, r
	}

	return http.NotFoundHandler(), r
}

// Use registers middleware.
func (mux *Mux) Use(middleware ...func(http.Handler) http.Handler) {
	mux.middleware = append(mux.middleware, middleware...)
}

func cleanPath(p string) string {
	if p == "" {
		return "/"
	}

	if p[0] != '/' {
		p = "/" + p
	}

	np := path.Clean(p)
	if p[len(p)-1] == '/' && np != "/" {
		np += "/"
	}
	return np
}
