package server

import "sync"

type hub struct {
	mu          sync.RWMutex
	subscribers map[chan map[string]any]struct{}
}

func newHub() *hub {
	return &hub{subscribers: make(map[chan map[string]any]struct{})}
}

func (h *hub) subscribe() (chan map[string]any, func()) {
	ch := make(chan map[string]any, 32)

	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()

	return ch, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, exists := h.subscribers[ch]; exists {
			delete(h.subscribers, ch)
			close(ch)
		}
	}
}

func (h *hub) broadcast(state map[string]any) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	delivered := 0
	for ch := range h.subscribers {
		select {
		case ch <- state:
			delivered++
		default:
			logger().Debug("dropping sync event for slow subscriber")
		}
	}
	return delivered
}
