package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"golang.org/x/oauth2"
)

// startProxy creates a persistent listener and accepts client connections in a loop.
// Each client connection triggers a realm dial and relay session. The listener stays
// alive across sessions so the port isn't released and rebound.
func startProxy(ctx context.Context, listenAddr, inviteCode string, tokenSource oauth2.TokenSource, state *GameState) {
	cfg := minecraft.ListenConfig{
		AuthenticationDisabled: true,
		StatusProvider:         minecraft.NewStatusProvider("Burnodd Realm Proxy", "Gophertunnel"),
		AcceptedProtocols:      nil,
		AllowUnknownPackets:    true,
		AllowInvalidPackets:    true,
		ErrorLog:               slog.Default(),
	}

	listener, err := cfg.Listen("raknet", listenAddr)
	if err != nil {
		slog.Error("failed to start listener", "error", err)
		return
	}
	defer listener.Close()

	slog.Info("proxy listening", "address", listenAddr)

	// Close listener when context is cancelled
	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		state.SetStatus(StatusWaitingForClient)

		c, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			slog.Error("accept error", "error", err)
			continue
		}

		clientConn := c.(*minecraft.Conn)
		slog.Info("client connected", "remote", clientConn.RemoteAddr())

		if err := handleSession(ctx, clientConn, inviteCode, tokenSource, state); err != nil {
			slog.Error("session error", "error", err)
		}

		state.ClearConnections()
		state.SetStatus(StatusDisconnected)
		slog.Info("session ended, waiting for new client")
	}
}

// handleSession manages one client→realm relay session.
func handleSession(ctx context.Context, clientConn *minecraft.Conn, inviteCode string, tokenSource oauth2.TokenSource, state *GameState) error {
	state.SetStatus(StatusConnectingToRealm)

	// Resolve realm address
	realmAddr, err := resolveRealmAddress(ctx, tokenSource, inviteCode)
	if err != nil {
		clientConn.Close()
		return err
	}

	// Dial the realm
	dialer := minecraft.Dialer{
		TokenSource: tokenSource,
	}
	serverConn, err := dialer.DialContext(ctx, "raknet", realmAddr)
	if err != nil {
		clientConn.Close()
		return err
	}

	// Perform handshake: spawn the client and the server connection
	gd := serverConn.GameData()
	id := serverConn.IdentityData()

	errs := make(chan error, 2)
	go func() {
		errs <- clientConn.StartGame(gd)
	}()
	go func() {
		errs <- serverConn.DoSpawn()
	}()
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			serverConn.Close()
			clientConn.Close()
			return err
		}
	}

	slog.Info("connected to realm",
		"world", gd.WorldName,
		"player", id.DisplayName,
		"xuid", id.XUID,
	)

	// Initialize state
	state.SetConnections(serverConn, clientConn)
	state.SetIdentity(id.DisplayName, id.XUID, gd.EntityRuntimeID)
	state.InitFromGameData(gd)
	state.SetStatus(StatusConnected)

	// Start PlayerAuthInput tick loop to keep the realm connection alive
	sessionCtx, sessionCancel := context.WithCancel(ctx)
	defer sessionCancel()

	go playerAuthInputLoop(sessionCtx, serverConn, gd)

	// Relay packets bidirectionally with interception
	done := make(chan struct{}, 2)

	// client → server
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			pk, err := clientConn.ReadPacket()
			if err != nil {
				return
			}
			interceptClientPacket(pk, state)
			if err := serverConn.WritePacket(pk); err != nil {
				return
			}
		}
	}()

	// server → client
	go func() {
		defer func() { done <- struct{}{} }()
		for {
			pk, err := serverConn.ReadPacket()
			if err != nil {
				return
			}
			interceptServerPacket(pk, state)
			if err := clientConn.WritePacket(pk); err != nil {
				return
			}
		}
	}()

	// Wait for either relay to finish (disconnect)
	select {
	case <-done:
	case <-ctx.Done():
	}

	sessionCancel()
	serverConn.Close()
	clientConn.Close()

	return nil
}

// playerAuthInputLoop sends PlayerAuthInput packets every tick (50ms) to keep
// the Realm treating us as an active player. Without this, Realms silently
// drops chat/command packets.
func playerAuthInputLoop(ctx context.Context, conn *minecraft.Conn, gd minecraft.GameData) {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	var tick uint64
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tick++
			conn.WritePacket(&packet.PlayerAuthInput{
				Position:         gd.PlayerPosition,
				Pitch:            gd.Pitch,
				Yaw:              gd.Yaw,
				HeadYaw:          gd.Yaw,
				InputData:        protocol.NewBitset(packet.PlayerAuthInputBitsetSize),
				Tick:             tick,
				InputMode:        packet.InputModeMouse,
				PlayMode:         packet.PlayModeNormal,
				InteractionModel: packet.InteractionModelCrosshair,
			})
		}
	}
}
