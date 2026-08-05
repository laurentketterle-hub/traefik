package server

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
)

// RouterConfig represents the configuration for a router.
type RouterConfig struct {
	Path         string
	Middleware   string
	ResponseText string
}

// MiddlewareConfig represents the configuration for a middleware.
type MiddlewareConfig struct {
	HeaderName  string
	HeaderValue string
}

// Configuration represents a complete, self-contained dynamic configuration snapshot.
type Configuration struct {
	Routers     map[string]RouterConfig
	Middlewares map[string]MiddlewareConfig
}

// serverState bundles the active handler and its corresponding configuration
// as a single immutable snapshot. This ensures that the handler chain and
// the configuration reported by the API are always consistent.
type serverState struct {
	config  Configuration
	handler http.Handler
}

// EntryPoint represents an entrypoint that serves HTTP requests.
// The active state (handler + config) is swapped atomically.
type EntryPoint struct {
	state atomic.Value // holds *serverState
}

func (e *EntryPoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	st := e.state.Load()
	if st != nil {
		st.(*serverState).handler.ServeHTTP(w, r)
	} else {
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

// currentConfig returns the config that is currently active (matching the handler).
func (e *EntryPoint) currentConfig() Configuration {
	st := e.state.Load()
	if st != nil {
		return st.(*serverState).config
	}
	return Configuration{}
}

// Server manages the entrypoints and configuration updates.
type Server struct {
	configurationChan chan Configuration
	entryPoints       map[string]*EntryPoint
	mu                sync.RWMutex
	configErrors      []string
}

func NewServer() *Server {
	ep := &EntryPoint{}
	// Initialize with a consistent default state using *serverState.
	ep.state.Store(&serverState{
		config: Configuration{},
		handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Not Found", http.StatusNotFound)
		}),
	})

	s := &Server{
		configurationChan: make(chan Configuration, 100),
		entryPoints: map[string]*EntryPoint{
			"web": ep,
		},
	}
	return s
}

func (s *Server) Start(ctx context.Context) {
	go s.watcher(ctx)
}

func (s *Server) watcher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case config := <-s.configurationChan:
			s.switchConfigs(config)
		}
	}
}

func (s *Server) GetConfigurationChan() chan<- Configuration {
	return s.configurationChan
}

// buildHandlerChain builds an http.Handler from the configuration snapshot.
// It returns the handler and a list of problems (duplicate paths, empty paths, etc.)
// so the watcher never panics on malformed config.
func (s *Server) buildHandlerChain(config Configuration) (http.Handler, []string) {
	var problems []string
	mux := http.NewServeMux()
	seen := make(map[string]string) // path -> router name

	// Sort router names for deterministic ordering
	names := make([]string, 0, len(config.Routers))
	for name := range config.Routers {
		names = append(names, name)
	}
	// Simple sort for small sets
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	for _, name := range names {
		cfg := config.Routers[name]
		if cfg.Path == "" {
			problems = append(problems, "router \""+name+"\" has no path, skipped")
			continue
		}
		if existing, ok := seen[cfg.Path]; ok {
			problems = append(problems, "router \""+name+"\" conflicts with \""+existing+"\" on path "+cfg.Path+", skipped")
			continue
		}
		seen[cfg.Path] = name

		// Capture by copy for closure
		routerCfg := cfg
		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(routerCfg.ResponseText))
		})

		if routerCfg.Middleware != "" {
			if mwCfg, ok := config.Middlewares[routerCfg.Middleware]; ok {
				handler = buildMiddleware(mwCfg, handler)
			}
		}
		mux.Handle(routerCfg.Path, handler)
	}

	return mux, problems
}

// switchConfigs builds a complete, self-consistent serverState from the
// supplied Configuration and swaps it in atomically. The handler and config
// are always swapped together, eliminating the window where they could be
// out of sync.
func (s *Server) switchConfigs(config Configuration) {
	// Build the complete handler chain, handling malformed config gracefully.
	mux, problems := s.buildHandlerChain(config)

	// Build the complete state snapshot and swap it atomically.
	// This bundles config + handler together so they never diverge.
	newState := &serverState{
		config:  config,
		handler: mux,
	}

	s.mu.Lock()
	s.configErrors = problems
	s.entryPoints["web"].state.Store(newState)
	s.mu.Unlock()
}

// buildMiddleware creates a middleware handler that sets the configured header
// before passing the request to the next handler.
func buildMiddleware(cfg MiddlewareConfig, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(cfg.HeaderName, cfg.HeaderValue)
		next.ServeHTTP(w, r)
	})
}

func (s *Server) GetEntryPoint(name string) *EntryPoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entryPoints[name]
}

// GetConfig returns the configuration that is currently active and consistent
// with the handler chain.
func (s *Server) GetConfig() Configuration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ep := s.entryPoints["web"]
	if ep == nil {
		return Configuration{}
	}
	return ep.currentConfig()
}

// GetConfigErrors returns errors from the last configuration build.
func (s *Server) GetConfigErrors() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]string{}, s.configErrors...)
}
