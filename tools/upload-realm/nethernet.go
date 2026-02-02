package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"

	nethernet "github.com/df-mc/go-nethernet"
	"github.com/sandertv/gophertunnel/minecraft"
)

// nethernetState holds the MCToken needed for signaling authentication.
// It is set before dialing and read by the Network implementation.
var nethernetState struct {
	mu      sync.Mutex
	mcToken string
}

// SetNetherNetToken sets the MCToken used for nethernet signaling authentication.
// Must be called before dialing with the "nethernet" network.
func SetNetherNetToken(token string) {
	nethernetState.mu.Lock()
	defer nethernetState.mu.Unlock()
	nethernetState.mcToken = token
}

func getNetherNetToken() string {
	nethernetState.mu.Lock()
	defer nethernetState.mu.Unlock()
	return nethernetState.mcToken
}

func init() {
	minecraft.RegisterNetwork("nethernet", func(l *slog.Logger) minecraft.Network {
		return &NetherNetNetwork{log: l}
	})
}

// NetherNetNetwork implements minecraft.Network for the NetherNet protocol.
// It bridges go-nethernet's WebRTC-based Dialer with gophertunnel's Network interface.
type NetherNetNetwork struct {
	log *slog.Logger
}

// DialContext establishes a connection to a remote network identified by the
// given address (a network ID string). It creates a WebSocket signaling connection,
// then uses go-nethernet's Dialer to establish the WebRTC connection.
func (n *NetherNetNetwork) DialContext(ctx context.Context, address string) (net.Conn, error) {
	mcToken := getNetherNetToken()
	if mcToken == "" {
		return nil, errors.New("nethernet: MCToken not set (call SetNetherNetToken first)")
	}

	n.log.Info("establishing nethernet connection", "networkID", address)

	// Create WebSocket signaling connection
	signaling, err := NewWebSocketSignaling(mcToken, n.log)
	if err != nil {
		return nil, fmt.Errorf("nethernet signaling: %w", err)
	}

	// Use go-nethernet's Dialer to establish the WebRTC connection
	dialer := nethernet.Dialer{
		Log: n.log,
	}

	conn, err := dialer.DialContext(ctx, address, signaling)
	if err != nil {
		signaling.Close()
		return nil, fmt.Errorf("nethernet dial: %w", err)
	}

	n.log.Info("nethernet connection established", "networkID", address)
	return conn, nil
}

// PingContext is not supported for NetherNet. Returning an error causes
// gophertunnel's Dialer to skip the ping and call DialContext directly.
func (n *NetherNetNetwork) PingContext(ctx context.Context, address string) ([]byte, error) {
	return nil, errors.New("nethernet: ping not supported")
}

// Listen is not implemented for NetherNet (we only need client-side dialing).
func (n *NetherNetNetwork) Listen(address string) (minecraft.NetworkListener, error) {
	return nil, errors.New("nethernet: listen not implemented")
}
