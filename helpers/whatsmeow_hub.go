package helpers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"gorm.io/gorm"
)

// WhatsmeowMessageHandler is called whenever a text message arrives from any connected tenant.
// replyJID is the full WhatsApp JID string (e.g. "261xxx@lid" or "628xxx@s.whatsapp.net") to use for replies.
type WhatsmeowMessageHandler func(schema, from, name, text, replyJID string)

// whatsmeowDeviceSchema is the GORM model for the schema→JID mapping table.
type whatsmeowDeviceSchema struct {
	SchemaName  string    `gorm:"column:schema_name;primaryKey"`
	JID         string    `gorm:"column:jid;not null"`
	Phone       string    `gorm:"column:phone"`
	ConnectedAt time.Time `gorm:"column:connected_at"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (whatsmeowDeviceSchema) TableName() string { return "public.whatsmeow_device_schema" }

// WhatsmeowHub manages per-tenant whatsmeow connections.
// One *whatsmeow.Client is maintained per tenant schema.
type WhatsmeowHub struct {
	mu        sync.RWMutex
	clients   map[string]*whatsmeow.Client
	phones    map[string]string
	container *sqlstore.Container
	db        *gorm.DB
	handler   WhatsmeowMessageHandler
}

// NewWhatsmeowHub creates a hub that reuses the existing GORM database connection.
// whatsmeow session tables are created automatically via sqlstore.Upgrade.
func NewWhatsmeowHub(db *gorm.DB) (*WhatsmeowHub, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB from gorm: %w", err)
	}
	dbLog := waLog.Stdout("whatsmeow-store", "WARN", true)
	container := sqlstore.NewWithDB(sqlDB, "postgres", dbLog)
	if err := container.Upgrade(context.Background()); err != nil {
		return nil, fmt.Errorf("whatsmeow sqlstore upgrade: %w", err)
	}
	return &WhatsmeowHub{
		clients:   make(map[string]*whatsmeow.Client),
		phones:    make(map[string]string),
		container: container,
		db:        db,
	}, nil
}

// SetMessageHandler registers the callback that handles incoming messages.
// Must be set before Start is called.
func (h *WhatsmeowHub) SetMessageHandler(fn WhatsmeowMessageHandler) {
	h.handler = fn
}

// Start reconnects all previously saved tenant sessions in the background.
func (h *WhatsmeowHub) Start() {
	var mappings []whatsmeowDeviceSchema
	if err := h.db.Find(&mappings).Error; err != nil {
		log.Printf("[WhatsmeowHub] failed to load saved sessions: %v", err)
		return
	}
	for _, m := range mappings {
		go h.reconnectSession(m.SchemaName, m.JID, m.Phone)
	}
}

func (h *WhatsmeowHub) reconnectSession(schema, jidStr, phone string) {
	ctx := context.Background()
	jid, err := types.ParseJID(jidStr)
	if err != nil {
		log.Printf("[WhatsmeowHub] invalid JID for schema=%s: %v", schema, err)
		return
	}
	device, err := h.container.GetDevice(ctx, jid)
	if err != nil || device == nil {
		log.Printf("[WhatsmeowHub] device not found for schema=%s jid=%s", schema, jidStr)
		return
	}
	client := h.buildClient(schema, device)

	h.mu.Lock()
	h.clients[schema] = client
	h.phones[schema] = phone
	h.mu.Unlock()

	if err := client.Connect(); err != nil {
		log.Printf("[WhatsmeowHub] reconnect failed for schema=%s: %v", schema, err)
	}
}

func (h *WhatsmeowHub) buildClient(schema string, device *store.Device) *whatsmeow.Client {
	logger := waLog.Stdout("WA:"+schema, "WARN", true)
	client := whatsmeow.NewClient(device, logger)
	client.AddEventHandler(h.makeEventHandler(schema))
	return client
}

func (h *WhatsmeowHub) makeEventHandler(schema string) func(interface{}) {
	return func(rawEvt interface{}) {
		switch evt := rawEvt.(type) {
		case *events.Message:
			if evt.Info.IsFromMe || evt.Info.IsGroup {
				return
			}
			text := extractWAMessageText(evt.Message)
			if text == "" {
				return
			}
			from := evt.Info.Chat.User
			replyJID := evt.Info.Chat.String()
			name := evt.Info.PushName
			log.Printf("[WhatsmeowHub] schema=%s from=%s replyJID=%s text=%.40s", schema, from, replyJID, text)
			if h.handler != nil {
				h.handler(schema, from, name, text, replyJID)
			}
		case *events.Connected:
			log.Printf("[WhatsmeowHub] ✅ Connected for schema=%s", schema)
		case *events.Disconnected:
			log.Printf("[WhatsmeowHub] Disconnected for schema=%s", schema)
		case *events.LoggedOut:
			log.Printf("[WhatsmeowHub] 🔴 LoggedOut for schema=%s — removing session", schema)
			h.cleanupInMemory(schema)
		}
	}
}

func extractWAMessageText(msg *waE2E.Message) string {
	if msg == nil {
		return ""
	}
	if t := msg.GetConversation(); t != "" {
		return t
	}
	if ext := msg.GetExtendedTextMessage(); ext != nil {
		return ext.GetText()
	}
	return ""
}

// ConnectWithQR starts a new QR-based login for schema.
// Returns a channel that emits QRChannelItem events until the connection
// succeeds ("success"), times out ("timeout"), or the context is cancelled.
func (h *WhatsmeowHub) ConnectWithQR(ctx context.Context, schema string) (<-chan whatsmeow.QRChannelItem, error) {
	_ = h.Disconnect(schema) // clean up any existing session first

	device := h.container.NewDevice()
	client := h.buildClient(schema, device)

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return nil, fmt.Errorf("get QR channel: %w", err)
	}

	h.mu.Lock()
	h.clients[schema] = client
	h.mu.Unlock()

	go func() {
		if err := client.Connect(); err != nil {
			log.Printf("[WhatsmeowHub] connect error schema=%s: %v", schema, err)
		}
	}()

	out := make(chan whatsmeow.QRChannelItem, 8)
	go func() {
		defer close(out)
		for item := range qrChan {
			out <- item
			if item.Event == "success" && client.Store.ID != nil {
				phone := client.Store.ID.User
				h.mu.Lock()
				h.phones[schema] = phone
				h.mu.Unlock()
				h.saveSession(schema, client.Store.ID.String(), phone)
			}
		}
	}()

	return out, nil
}

func (h *WhatsmeowHub) saveSession(schema, jid, phone string) {
	mapping := whatsmeowDeviceSchema{
		SchemaName:  schema,
		JID:         jid,
		Phone:       phone,
		ConnectedAt: time.Now(),
	}
	h.db.Save(&mapping)
	log.Printf("[WhatsmeowHub] session saved schema=%s phone=%s", schema, phone)
}

func (h *WhatsmeowHub) cleanupInMemory(schema string) {
	h.mu.Lock()
	delete(h.clients, schema)
	delete(h.phones, schema)
	h.mu.Unlock()
	h.db.Where("schema_name = ?", schema).Delete(&whatsmeowDeviceSchema{})
}

// Disconnect logs out and removes the whatsmeow session for a schema.
func (h *WhatsmeowHub) Disconnect(schema string) error {
	h.mu.Lock()
	client, ok := h.clients[schema]
	delete(h.clients, schema)
	delete(h.phones, schema)
	h.mu.Unlock()

	if ok && client != nil {
		client.Disconnect()
		if client.Store.ID != nil {
			if err := client.Store.Delete(context.Background()); err != nil {
				log.Printf("[WhatsmeowHub] delete device error schema=%s: %v", schema, err)
			}
		}
	}
	h.db.Where("schema_name = ?", schema).Delete(&whatsmeowDeviceSchema{})
	return nil
}

// IsConnected returns true if the schema has an active whatsmeow connection.
func (h *WhatsmeowHub) IsConnected(schema string) bool {
	h.mu.RLock()
	client, ok := h.clients[schema]
	h.mu.RUnlock()
	return ok && client != nil && client.IsConnected()
}

// GetPhone returns the connected phone number for a schema, or "" if not connected.
func (h *WhatsmeowHub) GetPhone(schema string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.phones[schema]
}

// GetSender returns a WhatsAppSender for the schema, or nil if not connected.
func (h *WhatsmeowHub) GetSender(schema string) WhatsAppSender {
	h.mu.RLock()
	client, ok := h.clients[schema]
	h.mu.RUnlock()
	if !ok || client == nil || !client.IsConnected() {
		return nil
	}
	return &WhatsmeowSender{client: client}
}

// WhatsmeowSender implements WhatsAppSender using a whatsmeow client.
type WhatsmeowSender struct {
	client *whatsmeow.Client
}

func (s *WhatsmeowSender) SendMessage(to, text string) error {
	if !s.client.IsConnected() {
		return fmt.Errorf("whatsmeow client not connected")
	}

	var jid types.JID
	if strings.Contains(to, "@") {
		// Full JID provided (e.g. "261xxx@lid" or "628xxx@s.whatsapp.net")
		parsed, err := types.ParseJID(to)
		if err != nil {
			return fmt.Errorf("invalid JID %q: %w", to, err)
		}
		jid = parsed
	} else {
		to = strings.TrimPrefix(to, "+")
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	log.Printf("[WhatsmeowSender] sending to JID: %s", jid)
	msg := &waE2E.Message{Conversation: proto.String(text)}
	sendResp, err := s.client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		log.Printf("[WhatsmeowSender] SendMessage FAILED to %s: %v", jid, err)
	} else {
		log.Printf("[WhatsmeowSender] SendMessage OK to %s msgID=%s", jid, sendResp.ID)
	}
	return err
}

func (s *WhatsmeowSender) SendTemplateMessage(to, templateName, languageCode string, bodyParams []string) error {
	// whatsmeow does not support Meta-style templates; send as plain text
	return s.SendMessage(to, strings.Join(bodyParams, "\n"))
}

func (s *WhatsmeowSender) SendImageMessage(to, imageURL, caption string) error {
	if !s.client.IsConnected() {
		return fmt.Errorf("whatsmeow client not connected")
	}

	resp, err := http.Get(imageURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read image body: %w", err)
	}

	uploaded, err := s.client.Upload(context.Background(), imageBytes, whatsmeow.MediaImage)
	if err != nil {
		return fmt.Errorf("failed to upload image to whatsapp: %w", err)
	}

	var jid types.JID
	if strings.Contains(to, "@") {
		parsed, err := types.ParseJID(to)
		if err != nil {
			return fmt.Errorf("invalid JID %q: %w", to, err)
		}
		jid = parsed
	} else {
		to = strings.TrimPrefix(to, "+")
		jid = types.NewJID(to, types.DefaultUserServer)
	}

	mimetype := http.DetectContentType(imageBytes)
	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimetype),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uploaded.FileLength),
		},
	}

	sendResp, err := s.client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		log.Printf("[WhatsmeowSender] SendImageMessage FAILED to %s: %v", jid, err)
	} else {
		log.Printf("[WhatsmeowSender] SendImageMessage OK to %s msgID=%s", jid, sendResp.ID)
	}
	return err
}
