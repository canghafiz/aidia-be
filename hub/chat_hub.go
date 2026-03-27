package hub

import (
	"sync"
)

// ChatHub manages SSE clients per tenant and guest
type ChatHub struct {
	mu      sync.RWMutex
	clients map[string]map[chan string]struct{} // key: tenant_id:guest_id
}

var chatHub = &ChatHub{
	clients: make(map[string]map[chan string]struct{}),
}

// GetChatHub returns the singleton chat hub instance
func GetChatHub() *ChatHub {
	return chatHub
}

// Subscribe subscribes to SSE updates for a specific guest
func (h *ChatHub) Subscribe(tenantID, guestID string) chan string {
	ch := make(chan string, 10)
	h.mu.Lock()
	defer h.mu.Unlock()

	key := tenantID
	if guestID != "" {
		key = tenantID + ":" + guestID
	}
	
	if h.clients[key] == nil {
		h.clients[key] = make(map[chan string]struct{})
	}
	h.clients[key][ch] = struct{}{}
	return ch
}

// SubscribeToTenant subscribes to all guest updates for a tenant
func (h *ChatHub) SubscribeToTenant(tenantID string) chan string {
	ch := make(chan string, 10)
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.clients[tenantID] == nil {
		h.clients[tenantID] = make(map[chan string]struct{})
	}
	h.clients[tenantID][ch] = struct{}{}
	return ch
}

// Unsubscribe unsubscribes from SSE updates
func (h *ChatHub) Unsubscribe(tenantID, guestID string, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	key := tenantID
	if guestID != "" {
		key = tenantID + ":" + guestID
	}
	
	if h.clients[key] != nil {
		delete(h.clients[key], ch)
		if len(h.clients[key]) == 0 {
			delete(h.clients, key)
		}
	}
	close(ch)
}

// BroadcastToGuest sends update to specific guest
func (h *ChatHub) BroadcastToGuest(tenantID, guestID, data string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	key := tenantID + ":" + guestID
	for ch := range h.clients[key] {
		select {
		case ch <- data:
		default:
			// Channel full, skip
		}
	}
}

// BroadcastToTenant sends update to all guests in a tenant
func (h *ChatHub) BroadcastToTenant(tenantID, data string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for key, clients := range h.clients {
		if key == tenantID || len(key) > len(tenantID) && key[:len(tenantID)+1] == tenantID+":" {
			for ch := range clients {
				select {
				case ch <- data:
				default:
				}
			}
		}
	}
}
