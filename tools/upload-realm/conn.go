package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/realms"
)

// RealmConn bundles a Minecraft connection with its context and cancel function.
type RealmConn struct {
	Conn   *minecraft.Conn
	Ctx    context.Context
	Cancel context.CancelFunc
}

// dialRealm performs the full connection sequence: invite lookup, auth, realm
// discovery, dialing, and spawning. The caller must close Conn and call Cancel.
func dialRealm() (*RealmConn, error) {
	inviteCode, err := getRealmInvite()
	if err != nil {
		return nil, err
	}

	tokenSource, err := getTokenSource()
	if err != nil {
		return nil, fmt.Errorf("auth error: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	realmsClient := realms.NewClient(tokenSource, nil)
	fmt.Println("Looking up Realm...")
	realm, err := realmsClient.Realm(ctx, inviteCode)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("realm lookup error: %w", err)
	}

	address, err := realm.Address(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("realm address error: %w", err)
	}

	fmt.Printf("Connecting to %s (%s)...\n", realm.Name, address)

	dialer := minecraft.Dialer{
		TokenSource: tokenSource,
	}

	conn, err := dialer.Dial("raknet", address)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("connection error: %w", err)
	}

	if err := conn.DoSpawn(); err != nil {
		conn.Close()
		cancel()
		return nil, fmt.Errorf("spawn error: %w", err)
	}

	gd := conn.GameData()
	fmt.Printf("Spawned! World: %s, Gamemode: %d, Position: (%.1f, %.1f, %.1f)\n",
		gd.WorldName, gd.PlayerGameMode,
		gd.PlayerPosition.X(), gd.PlayerPosition.Y(), gd.PlayerPosition.Z())

	return &RealmConn{Conn: conn, Ctx: ctx, Cancel: cancel}, nil
}

// setupSignalHandler arranges for cancel to be called on SIGINT or SIGTERM.
func setupSignalHandler(cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, shutting down...")
		cancel()
	}()
}
