package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"golang.org/x/oauth2"
)

const (
	// The relying party for Xbox Live session directory API (MPSD).
	// This is "https://xboxlive.com/" not the directory hostname itself.
	sessionDirectoryRP = "https://xboxlive.com/"
	xblContractVersion = "107"
)

// ResolveNetherNetID resolves an MPSD session handle UUID (from the Realms API)
// to the uint64 WebRTCNetworkId needed for NetherNet signaling.
func ResolveNetherNetID(ctx context.Context, tokenSource oauth2.TokenSource, handleUUID string) (string, error) {
	// Parse and validate the UUID
	handleID, err := uuid.Parse(handleUUID)
	if err != nil {
		return "", fmt.Errorf("invalid handle UUID %q: %w", handleUUID, err)
	}

	// Get an XBL token for the session directory
	liveToken, err := tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("get live token: %w", err)
	}
	xblToken, err := auth.RequestXBLToken(ctx, liveToken, sessionDirectoryRP)
	if err != nil {
		return "", fmt.Errorf("get XBL token for session directory: %w", err)
	}

	// Try GET on the handle to retrieve session info with custom properties
	networkID, err := getHandleNetworkID(ctx, xblToken, handleID)
	if err != nil {
		// Fallback: try joining the session via PUT to get full session data
		fmt.Printf("Handle GET failed (%v), trying session join...\n", err)
		networkID, err = joinSessionNetworkID(ctx, xblToken, handleID)
		if err != nil {
			return "", fmt.Errorf("resolve network ID: %w", err)
		}
	}

	return networkID, nil
}

// getHandleNetworkID tries to GET the handle directly to read custom properties.
func getHandleNetworkID(ctx context.Context, xblToken *auth.XBLToken, handleID uuid.UUID) (string, error) {
	url := fmt.Sprintf("https://sessiondirectory.xboxlive.com/handles/%s", handleID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	xblToken.SetAuthHeader(req)
	req.Header.Set("X-Xbl-Contract-Version", xblContractVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET handle: %s (body: %s)", resp.Status, truncate(string(body), 200))
	}

	fmt.Printf("Handle response: %s\n", truncate(string(body), 500))
	return extractNetworkID(body)
}

// joinSessionNetworkID joins the session via the handle and reads the WebRTCNetworkId
// from the session's custom properties.
func joinSessionNetworkID(ctx context.Context, xblToken *auth.XBLToken, handleID uuid.UUID) (string, error) {
	url := fmt.Sprintf("https://sessiondirectory.xboxlive.com/handles/%s/session", handleID)

	// Minimal session description to join
	body := `{"members":{"me":{"constants":{"system":{"initialize":true}},"properties":{"system":{"active":true,"connection":""}}}}}`

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, strings.NewReader(body))
	if err != nil {
		return "", err
	}
	xblToken.SetAuthHeader(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Xbl-Contract-Version", xblContractVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("PUT session: %s (body: %s)", resp.Status, truncate(string(respBody), 200))
	}

	fmt.Printf("Session response: %s\n", truncate(string(respBody), 500))
	return extractNetworkID(respBody)
}

// extractNetworkID parses a JSON response and looks for WebRTCNetworkId in
// various locations: custom properties, session properties, supported connections.
func extractNetworkID(data []byte) (string, error) {
	// Try to find WebRTCNetworkId in the top-level custom properties (handle response)
	var handleResp struct {
		CustomProperties json.RawMessage `json:"customProperties"`
	}
	if err := json.Unmarshal(data, &handleResp); err == nil && len(handleResp.CustomProperties) > 0 {
		if id, err := parseNetworkIDFromCustom(handleResp.CustomProperties); err == nil {
			return id, nil
		}
	}

	// Try session properties (session response)
	var sessionResp struct {
		Properties *struct {
			Custom json.RawMessage `json:"custom"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(data, &sessionResp); err == nil && sessionResp.Properties != nil && len(sessionResp.Properties.Custom) > 0 {
		if id, err := parseNetworkIDFromCustom(sessionResp.Properties.Custom); err == nil {
			return id, nil
		}
	}

	// Try member properties (in case the network ID is in member custom data)
	var memberResp struct {
		Members map[string]struct {
			Properties *struct {
				Custom json.RawMessage `json:"custom"`
			} `json:"properties"`
		} `json:"members"`
	}
	if err := json.Unmarshal(data, &memberResp); err == nil {
		for _, member := range memberResp.Members {
			if member.Properties != nil && len(member.Properties.Custom) > 0 {
				if id, err := parseNetworkIDFromCustom(member.Properties.Custom); err == nil {
					return id, nil
				}
			}
		}
	}

	return "", fmt.Errorf("WebRTCNetworkId not found in response")
}

// parseNetworkIDFromCustom parses WebRTCNetworkId from a custom properties JSON blob.
func parseNetworkIDFromCustom(data json.RawMessage) (string, error) {
	var props struct {
		WebRTCNetworkID json.Number `json:"WebRTCNetworkId"`
		SupportedConns  []struct {
			NetherNetID     json.Number `json:"NetherNetId"`
			WebRTCNetworkID json.Number `json:"WebRTCNetworkId"`
		} `json:"SupportedConnections"`
	}
	if err := json.Unmarshal(data, &props); err != nil {
		return "", err
	}

	// Try direct WebRTCNetworkId field
	if props.WebRTCNetworkID != "" && props.WebRTCNetworkID != "0" {
		// Validate it's a real number
		if _, err := strconv.ParseUint(string(props.WebRTCNetworkID), 10, 64); err == nil {
			fmt.Printf("Found WebRTCNetworkId: %s\n", props.WebRTCNetworkID)
			return string(props.WebRTCNetworkID), nil
		}
	}

	// Try SupportedConnections
	for _, conn := range props.SupportedConns {
		if conn.NetherNetID != "" && conn.NetherNetID != "0" {
			if _, err := strconv.ParseUint(string(conn.NetherNetID), 10, 64); err == nil {
				fmt.Printf("Found NetherNetId from SupportedConnections: %s\n", conn.NetherNetID)
				return string(conn.NetherNetID), nil
			}
		}
		if conn.WebRTCNetworkID != "" && conn.WebRTCNetworkID != "0" {
			if _, err := strconv.ParseUint(string(conn.WebRTCNetworkID), 10, 64); err == nil {
				fmt.Printf("Found WebRTCNetworkId from SupportedConnections: %s\n", conn.WebRTCNetworkID)
				return string(conn.WebRTCNetworkID), nil
			}
		}
	}

	return "", fmt.Errorf("no valid network ID in custom properties")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
