package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type App struct {
	hub    *hub
	store  *stateStore
	state  *runtimeState
	config Config
}

type runtimeState struct {
	mu     sync.RWMutex
	state  map[string]any
	tokens map[string]struct{}
}

func newApp(cfg Config) (*App, error) {
	store := newStateStore(cfg.ConfigFile)
	persisted, err := store.Load()
	if err != nil {
		return nil, err
	}

	return &App{
		hub:    newHub(),
		store:  store,
		state:  newRuntimeState(persisted),
		config: cfg,
	}, nil
}

func newRuntimeState(persisted State) *runtimeState {
	state := &runtimeState{}
	state.Replace(persisted)
	return state
}

func (s *runtimeState) Replace(persisted State) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.state = cloneJSONObject(persisted.State)
	s.tokens = tokenSliceToSet(persisted.Tokens)
}

func (s *runtimeState) Snapshot() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneJSONObject(s.state)
}

func (s *runtimeState) Allows(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.tokens[token]
	return ok
}

func (a *App) watchLocalState() {
	go func() {
		for {
			waitForFileChange(a.config.ClaudeSettingsFile, a.config.ClaudeCredentialsFile)

			recipients, changed, err := a.syncLocalState()
			if err != nil {
				logger().Error("failed to sync local Claude state", "error", err)
				continue
			}
			if !changed {
				continue
			}

			logger().Debug("broadcast local Claude state update", "recipients", recipients)
		}
	}()
}

func (a *App) syncLocalState() (int, bool, error) {
	persisted, changed, err := a.store.SyncStateFromClaudeFiles(a.config.ClaudeSettingsFile, a.config.ClaudeCredentialsFile)
	if err != nil {
		return 0, false, err
	}
	if !changed {
		return 0, false, nil
	}

	a.state.Replace(persisted)
	return a.broadcastState(), true, nil
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/events", a.handleEvents)
	mux.HandleFunc("/sync", a.handleSync)
	return withLogging(mux)
}

func (a *App) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) handleEvents(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.authorizeRequest(r); !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	writeSSEHeaders(w)

	ctx := r.Context()
	ch, unsubscribe := a.hub.subscribe()
	defer unsubscribe()

	sendSSE(w, "connected", map[string]string{"status": "connected"})
	sendSSE(w, "config-sync", a.state.Snapshot())
	flusher.Flush()

	keepAlive := time.NewTicker(20 * time.Second)
	defer keepAlive.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case state, ok := <-ch:
			if !ok {
				return
			}
			sendSSE(w, "config-sync", state)
			flusher.Flush()
		case <-keepAlive.C:
			sendSSEComment(w, "keep-alive")
			flusher.Flush()
		}
	}
}

func (a *App) handleSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	token, ok := a.authorizeRequest(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	nextState, err := decodeJSONObjectRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if a.config.WatchLocalState {
		if err := writeClaudeStateFiles(a.config.ClaudeSettingsFile, a.config.ClaudeCredentialsFile, nextState); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update local Claude state")
			return
		}
	}

	persisted, err := a.store.ReplaceState(nextState)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to persist state")
		return
	}

	a.state.Replace(persisted)
	recipients := a.broadcastState()
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status":     "broadcast",
		"token":      token,
		"recipients": recipients,
	})
}

func (a *App) authorizeRequest(r *http.Request) (string, bool) {
	token := bearerToken(r)
	if token == "" {
		return "", false
	}
	return token, a.state.Allows(token)
}

func (a *App) broadcastState() int {
	return a.hub.broadcast(a.state.Snapshot())
}

func decodeJSONObjectRequest(r *http.Request) (map[string]any, error) {
	decoder := json.NewDecoder(r.Body)

	var payload any
	if err := decoder.Decode(&payload); err != nil {
		return nil, fmt.Errorf("invalid JSON body")
	}
	if decoder.More() {
		return nil, fmt.Errorf("invalid JSON body")
	}

	return ensureJSONObject(payload)
}

func tokenSliceToSet(tokens []string) map[string]struct{} {
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		set[token] = struct{}{}
	}
	return set
}

func cloneJSONObject(value map[string]any) map[string]any {
	cloned := make(map[string]any, len(value))
	for key, item := range value {
		cloned[key] = cloneJSONValue(item)
	}
	return cloned
}

func cloneJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneJSONObject(typed)
	case []any:
		cloned := make([]any, 0, len(typed))
		for _, item := range typed {
			cloned = append(cloned, cloneJSONValue(item))
		}
		return cloned
	default:
		return typed
	}
}
