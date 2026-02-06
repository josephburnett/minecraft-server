package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/realms"
	"golang.org/x/oauth2"
)

// resolveRealmAddress looks up a Realm by invite code and returns its RakNet address.
func resolveRealmAddress(ctx context.Context, tokenSource oauth2.TokenSource, inviteCode string) (string, error) {
	client := realms.NewClient(tokenSource, nil)

	slog.Info("looking up realm...")
	realm, err := client.Realm(ctx, inviteCode)
	if err != nil {
		return "", fmt.Errorf("realm lookup error: %w", err)
	}

	slog.Info("found realm", "name", realm.Name, "id", realm.ID)

	// Retry loop: handles 503 (realm starting up) and NETHERNET UUID addresses.
	for attempt := range 10 {
		address, protocol, err := realmJoin(ctx, tokenSource, realm.ID)
		if err != nil {
			slog.Warn("realm join failed, retrying...", "error", err, "attempt", attempt+1)
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(3 * time.Second):
			}
			continue
		}

		slog.Info("realm join response", "address", address, "protocol", protocol, "attempt", attempt+1)

		if _, _, err := net.SplitHostPort(address); err == nil {
			return address, nil
		}

		slog.Warn("address not in host:port format (likely NETHERNET), retrying...", "address", address)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(3 * time.Second):
		}
	}

	return "", fmt.Errorf("realm address never resolved to host:port — realm may only support NETHERNET (WebRTC)")
}

// realmJoin calls the Realms API join endpoint directly and returns the address and protocol.
func realmJoin(ctx context.Context, tokenSource oauth2.TokenSource, realmID int) (address, protocol string, err error) {
	t, err := tokenSource.Token()
	if err != nil {
		return "", "", err
	}
	xbl, err := auth.RequestXBLToken(ctx, t, "https://pocket.realms.minecraft.net/")
	if err != nil {
		return "", "", err
	}

	url := fmt.Sprintf("https://pocket.realms.minecraft.net/worlds/%d/join", realmID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "MCPE/UWP")
	req.Header.Set("Client-Version", "1.10.1")
	xbl.SetAuthHeader(req)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	slog.Debug("realm join raw response", "status", resp.StatusCode, "body", string(body))

	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("realms API error %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Address         string `json:"address"`
		NetworkProtocol string `json:"networkProtocol"`
		PendingUpdate   bool   `json:"pendingUpdate"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", err
	}

	return data.Address, data.NetworkProtocol, nil
}

// realmJoinWithBackoff retries the join call on 503 errors.
func realmJoinWithBackoff(ctx context.Context, tokenSource oauth2.TokenSource, realmID int) (address, protocol string, err error) {
	delays := []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	for i := range 4 {
		address, protocol, err = realmJoin(ctx, tokenSource, realmID)
		if err == nil {
			return
		}
		if i < len(delays) {
			slog.Warn("realm join failed, retrying...", "error", err, "attempt", i+1)
			select {
			case <-ctx.Done():
				return "", "", ctx.Err()
			case <-time.After(delays[i]):
			}
		}
	}
	return
}
