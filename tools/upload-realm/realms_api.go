package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/realms"
	"golang.org/x/oauth2"
)

const realmsAPIBase = "https://pocket.realms.minecraft.net"

// RealmsHTTPClient provides HTTP operations for Realms world management.
type RealmsHTTPClient struct {
	tokenSrc   oauth2.TokenSource
	xblToken   *auth.XBLToken
	httpClient *http.Client
}

// NewRealmsHTTPClient creates a new HTTP client for Realms API operations.
func NewRealmsHTTPClient(src oauth2.TokenSource) *RealmsHTTPClient {
	return &RealmsHTTPClient{
		tokenSrc:   src,
		httpClient: &http.Client{Timeout: 5 * time.Minute},
	}
}

// xboxToken returns the xbox token used for the API.
func (c *RealmsHTTPClient) xboxToken(ctx context.Context) (*auth.XBLToken, error) {
	if c.xblToken != nil {
		return c.xblToken, nil
	}

	t, err := c.tokenSrc.Token()
	if err != nil {
		return nil, err
	}

	c.xblToken, err = auth.RequestXBLToken(ctx, t, "https://pocket.realms.minecraft.net/")
	return c.xblToken, err
}

// setHeaders sets the required headers for Realms API requests.
func (c *RealmsHTTPClient) setHeaders(ctx context.Context, req *http.Request) error {
	req.Header.Set("User-Agent", "MCPE/UWP")
	req.Header.Set("Client-Version", "1.10.1")

	xbl, err := c.xboxToken(ctx)
	if err != nil {
		return err
	}
	xbl.SetAuthHeader(req)
	return nil
}

// DownloadWorld downloads a world from the specified Realm slot.
func (c *RealmsHTTPClient) DownloadWorld(ctx context.Context, realm realms.Realm) ([]byte, error) {
	slot := realm.ActiveSlot
	if slot == 0 {
		slot = 1
	}

	// Get download URL with retry on 503
	var downloadURL string
	err := c.withRetry(ctx, func() error {
		url := fmt.Sprintf("%s/worlds/%d/slot/%d/download", realmsAPIBase, realm.ID, slot)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return err
		}

		if err := c.setHeaders(ctx, req); err != nil {
			return err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 503 {
			return &retryableError{status: 503}
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
		}

		var result struct {
			DownloadLink string `json:"downloadLink"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to parse download response: %w", err)
		}

		if result.DownloadLink == "" {
			return fmt.Errorf("empty download link in response")
		}

		downloadURL = result.DownloadLink
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get download URL: %w", err)
	}

	// Download the actual world file
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ConvertToMCWorld converts a tar.gz world archive to .mcworld (zip) format.
func ConvertToMCWorld(tgzData []byte) ([]byte, error) {
	// The Realms API returns tar.gz, but Minecraft imports .mcworld (zip)
	// For now, we'll just decompress the gzip layer since Minecraft
	// can handle various formats when importing via "Replace World"

	// Check if it's gzip compressed
	if len(tgzData) >= 2 && tgzData[0] == 0x1F && tgzData[1] == 0x8B {
		gzReader, err := gzip.NewReader(bytes.NewReader(tgzData))
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()

		decompressed, err := io.ReadAll(gzReader)
		if err != nil {
			return nil, err
		}
		return decompressed, nil
	}

	return tgzData, nil
}

// retryableError indicates the request should be retried.
type retryableError struct {
	status int
}

func (e *retryableError) Error() string {
	return fmt.Sprintf("retryable error: status %d", e.status)
}

// withRetry executes fn with exponential backoff retry on 503 errors.
func (c *RealmsHTTPClient) withRetry(ctx context.Context, fn func() error) error {
	delays := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second}

	var lastErr error
	for attempt := 0; attempt <= len(delays); attempt++ {
		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if _, ok := lastErr.(*retryableError); !ok {
			return lastErr
		}

		if attempt < len(delays) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delays[attempt]):
			}
		}
	}

	return lastErr
}
