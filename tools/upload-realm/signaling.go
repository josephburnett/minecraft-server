package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"

	nethernet "github.com/df-mc/go-nethernet"
	"github.com/gorilla/websocket"
)

const signalingURL = "wss://signal.franchise.minecraft-services.net/ws/v1.0/signaling"

// wsSignalMessage is a message sent to the signaling server.
type wsSignalMessage struct {
	Message string      `json:"Message"`
	To      json.Number `json:"To"`
	Type    int         `json:"Type"`
}

// wsSignalResponse is a message received from the signaling server.
type wsSignalResponse struct {
	Type    int    `json:"Type"`
	From    string `json:"From"`
	Message string `json:"Message"`
}

// WebSocketSignaling implements nethernet.Signaling over a WebSocket connection
// to the Minecraft signaling service.
type WebSocketSignaling struct {
	conn      *websocket.Conn
	networkID uint64
	mcToken   string
	log       *slog.Logger

	credentials *nethernet.Credentials
	credMu      sync.Mutex
	credReady   chan struct{}

	mu sync.Mutex
}

// NewWebSocketSignaling creates a new signaling connection. It connects to the
// signaling WebSocket and reads the initial credentials message.
func NewWebSocketSignaling(mcToken string, log *slog.Logger) (*WebSocketSignaling, error) {
	networkID := rand.Uint64()

	header := http.Header{}
	header.Set("Authorization", mcToken)

	url := fmt.Sprintf("%s/%d", signalingURL, networkID)
	if log != nil {
		log.Info("connecting to signaling server", "url", url)
	}

	conn, _, err := websocket.DefaultDialer.Dial(url, header)
	if err != nil {
		return nil, fmt.Errorf("signaling websocket dial: %w", err)
	}

	s := &WebSocketSignaling{
		conn:      conn,
		networkID: networkID,
		mcToken:   mcToken,
		log:       log,
		credReady: make(chan struct{}),
	}

	// Read the initial configuration message with TURN/STUN credentials
	if err := s.readInitialConfig(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("signaling read config: %w", err)
	}

	return s, nil
}

// readInitialConfig reads the first message from the signaling server,
// which contains TURN/STUN server credentials.
func (s *WebSocketSignaling) readInitialConfig() error {
	_, message, err := s.conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read initial message: %w", err)
	}

	var resp wsSignalResponse
	if err := json.Unmarshal(message, &resp); err != nil {
		return fmt.Errorf("unmarshal initial message: %w", err)
	}
	if resp.Type != 2 {
		return fmt.Errorf("expected config message type 2, got %d", resp.Type)
	}
	if resp.From != "Server" {
		return fmt.Errorf("expected config from Server, got %s", resp.From)
	}

	var creds nethernet.Credentials
	if err := json.Unmarshal([]byte(resp.Message), &creds); err != nil {
		return fmt.Errorf("unmarshal credentials: %w", err)
	}

	s.credMu.Lock()
	s.credentials = &creds
	s.credMu.Unlock()
	close(s.credReady)

	if s.log != nil {
		s.log.Info("received TURN/STUN credentials",
			"servers", len(creds.ICEServers),
			"expires_in", creds.ExpirationInSeconds)
	}

	return nil
}

// Signal sends a signal to a remote network via the signaling server.
func (s *WebSocketSignaling) Signal(signal *nethernet.Signal) error {
	msg := wsSignalMessage{
		Message: fmt.Sprintf("%s %d %s", signal.Type, signal.ConnectionID, signal.Data),
		To:      json.Number(signal.NetworkID),
		Type:    1,
	}

	b, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal signal: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.conn.WriteMessage(websocket.TextMessage, b); err != nil {
		return fmt.Errorf("write signal: %w", err)
	}

	if s.log != nil {
		s.log.Debug("sent signal", "type", signal.Type, "to", signal.NetworkID)
	}
	return nil
}

// Notify registers a notifier for incoming signals. It starts a goroutine that
// reads from the WebSocket and dispatches signals. Returns a stop function.
func (s *WebSocketSignaling) Notify(n nethernet.Notifier) (stop func()) {
	done := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)
		for {
			select {
			case <-done:
				return
			default:
			}

			_, message, err := s.conn.ReadMessage()
			if err != nil {
				select {
				case <-done:
					return
				default:
				}
				n.NotifyError(fmt.Errorf("signaling read: %w", err))
				return
			}

			var resp wsSignalResponse
			if err := json.Unmarshal(message, &resp); err != nil {
				if s.log != nil {
					s.log.Warn("failed to unmarshal signaling message", "error", err)
				}
				continue
			}

			// Type 2 messages are config updates, not signals
			if resp.Type == 2 {
				continue
			}

			// Parse the signal from the message
			parts := strings.SplitN(resp.Message, " ", 3)
			if len(parts) < 3 {
				if s.log != nil {
					s.log.Warn("malformed signal message", "message", resp.Message)
				}
				continue
			}

			connID, err := strconv.ParseUint(parts[1], 10, 64)
			if err != nil {
				if s.log != nil {
					s.log.Warn("invalid connection ID in signal", "raw", parts[1], "error", err)
				}
				continue
			}

			signal := &nethernet.Signal{
				Type:         parts[0],
				ConnectionID: connID,
				Data:         parts[2],
				NetworkID:    resp.From,
			}

			if s.log != nil {
				s.log.Debug("received signal", "type", signal.Type, "from", signal.NetworkID)
			}

			n.NotifySignal(signal)
		}
	}()

	return func() {
		close(done)
		s.conn.Close()
		<-stopped
	}
}

// Credentials returns the TURN/STUN credentials received from the signaling server.
func (s *WebSocketSignaling) Credentials(ctx context.Context) (*nethernet.Credentials, error) {
	select {
	case <-s.credReady:
		s.credMu.Lock()
		defer s.credMu.Unlock()
		return s.credentials, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// NetworkID returns our signaling network ID as a string.
func (s *WebSocketSignaling) NetworkID() string {
	return strconv.FormatUint(s.networkID, 10)
}

// PongData is a no-op for WebSocket signaling (used for LAN discovery).
func (s *WebSocketSignaling) PongData([]byte) {}

// Close closes the signaling WebSocket connection.
func (s *WebSocketSignaling) Close() error {
	return s.conn.Close()
}
