package hub

import "sync"

// KitchenHub manages SSE clients per schema
type KitchenHub struct {
	mu      sync.RWMutex
	clients map[string]map[chan string]struct{}
}

var kitchenHub = &KitchenHub{
	clients: make(map[string]map[chan string]struct{}),
}

func GetKitchenHub() *KitchenHub {
	return kitchenHub
}

func (h *KitchenHub) Subscribe(schema string) chan string {
	ch := make(chan string, 10)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[schema] == nil {
		h.clients[schema] = make(map[chan string]struct{})
	}
	h.clients[schema][ch] = struct{}{}
	return ch
}

func (h *KitchenHub) Unsubscribe(schema string, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[schema] != nil {
		delete(h.clients[schema], ch)
		if len(h.clients[schema]) == 0 {
			delete(h.clients, schema)
		}
	}
	close(ch)
}

func (h *KitchenHub) Broadcast(schema string, data string) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients[schema] {
		select {
		case ch <- data:
		default:
		}
	}
}
