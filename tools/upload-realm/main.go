package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
	"github.com/sandertv/gophertunnel/minecraft/realms"
	"golang.org/x/oauth2"
)

const tokenFile = ".realm-token"
const burnoddPackUUID = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"

func main() {
	chunksFile := flag.String("chunks", "structure.chunks", "Path to chunks file")
	flag.Parse()

	// Get realm invite code
	inviteCode, err := getRealmInvite()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Read chunks
	chunks, err := readChunks(*chunksFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Uploading %d chunks to Realm...\n", len(chunks))

	// Authenticate (with token caching)
	tokenSource, err := getTokenSource()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Auth error: %v\n", err)
		os.Exit(1)
	}

	// Get Realm address
	ctx := context.Background()
	realmsClient := realms.NewClient(tokenSource, nil)

	fmt.Println("Looking up Realm...")
	realm, err := realmsClient.Realm(ctx, inviteCode)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Realm lookup error: %v\n", err)
		os.Exit(1)
	}

	address, err := realm.Address(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Realm address error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Connecting to %s (%s)...\n", realm.Name, address)

	// Connect
	dialer := minecraft.Dialer{
		TokenSource: tokenSource,
	}

	conn, err := dialer.Dial("raknet", address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Connection error: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	// Spawn
	if err := conn.DoSpawn(); err != nil {
		fmt.Fprintf(os.Stderr, "Spawn error: %v\n", err)
		os.Exit(1)
	}

	// Check for burnodd_scripts pack
	packFound := false
	for _, pack := range conn.ResourcePacks() {
		if pack.UUID().String() == burnoddPackUUID {
			packFound = true
			fmt.Printf("Found behavior pack: %s\n", pack.Name())
			break
		}
	}
	if !packFound {
		fmt.Fprintf(os.Stderr, "Error: burnodd_scripts behavior pack not found on Realm!\n")
		fmt.Fprintf(os.Stderr, "Please add the pack to your Realm first.\n")
		os.Exit(1)
	}

	fmt.Println("Connected! Sending commands...")

	// Send commands
	for i, chunk := range chunks {
		cmd := fmt.Sprintf("/scriptevent burnodd:chunk %s", chunk)

		err := conn.WritePacket(&packet.CommandRequest{
			CommandLine: cmd,
			CommandOrigin: protocol.CommandOrigin{
				Origin: protocol.CommandOriginPlayer,
			},
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Command error: %v\n", err)
			break
		}

		if (i+1)%50 == 0 {
			fmt.Printf("Progress: %d / %d\n", i+1, len(chunks))
		}

		time.Sleep(50 * time.Millisecond)
	}

	fmt.Printf("Done! Sent %d commands.\n", len(chunks))
	time.Sleep(time.Second)
}

func getTokenSource() (oauth2.TokenSource, error) {
	// Try to load cached token
	token, err := loadToken()
	if err == nil {
		fmt.Println("Using cached authentication...")
		return auth.RefreshTokenSource(token), nil
	}

	// Request new token
	fmt.Println("Authenticating (check browser)...")
	token, err = auth.RequestLiveToken()
	if err != nil {
		return nil, err
	}

	// Save token for next time
	if err := saveToken(token); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not cache token: %v\n", err)
	}

	return auth.RefreshTokenSource(token), nil
}

func loadToken() (*oauth2.Token, error) {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return nil, err
	}
	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	return &token, nil
}

func saveToken(token *oauth2.Token) error {
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return os.WriteFile(tokenFile, data, 0600)
}

func getRealmInvite() (string, error) {
	// Check environment variable
	if code := os.Getenv("REALM_INVITE"); code != "" {
		return code, nil
	}

	// Check .realm-invite file
	exe, _ := os.Executable()
	dir := filepath.Dir(filepath.Dir(exe))
	inviteFile := filepath.Join(dir, ".realm-invite")

	// Also check current working directory
	if _, err := os.Stat(inviteFile); os.IsNotExist(err) {
		inviteFile = ".realm-invite"
	}

	data, err := os.ReadFile(inviteFile)
	if err != nil {
		return "", fmt.Errorf("no realm invite found; set REALM_INVITE or create .realm-invite file")
	}

	return strings.TrimSpace(string(data)), nil
}

func readChunks(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("chunks file not found: %s", path)
	}
	defer file.Close()

	var chunks []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			chunks = append(chunks, line)
		}
	}

	return chunks, scanner.Err()
}
