package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/auth"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"golang.org/x/oauth2"
)

const (
	playfabTitleID  = "20ca2"
	playfabSDK      = "XPlatCppSdk-3.6.19030404"
	playfabUA       = "libhttpclient/1.0.0.0"
	sessionStartURL = "https://authorization.franchise.minecraft-services.net/api/v1.0/session/start"
)

// PlayFabClient handles PlayFab and Minecraft service authentication to obtain
// an MCToken for NetherNet signaling.
type PlayFabClient struct {
	src    oauth2.TokenSource
	client *http.Client

	playFabID   string
	session     string
	entityToken string
	mcToken     string
}

// NewPlayFabClient authenticates with PlayFab and obtains an MCToken.
func NewPlayFabClient(src oauth2.TokenSource) (*PlayFabClient, error) {
	p := &PlayFabClient{
		src:    src,
		client: http.DefaultClient,
	}
	if err := p.loginWithXbox(); err != nil {
		return nil, fmt.Errorf("playfab login: %w", err)
	}
	if err := p.getEntityToken(); err != nil {
		return nil, fmt.Errorf("playfab entity token: %w", err)
	}
	if err := p.getMCToken(); err != nil {
		return nil, fmt.Errorf("mc token: %w", err)
	}
	return p, nil
}

// MCToken returns the Minecraft service token for signaling authentication.
func (p *PlayFabClient) MCToken() string {
	return p.mcToken
}

func (p *PlayFabClient) playfabRequest(endpoint string, body, result any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://%s.playfabapi.com/%s", playfabTitleID, endpoint)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", playfabUA)
	req.Header.Set("X-PlayFabSDK", playfabSDK)
	req.Header.Set("X-ReportErrorAsSuccess", "true")
	if p.entityToken != "" {
		req.Header.Set("X-EntityToken", p.entityToken)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(result)
}

func (p *PlayFabClient) externalRequest(url string, body, result any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", playfabUA)
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(result)
}

func (p *PlayFabClient) loginWithXbox() error {
	token, err := p.src.Token()
	if err != nil {
		return err
	}
	xbl, err := auth.RequestXBLToken(context.Background(), token, "rp://playfabapi.com/")
	if err != nil {
		return err
	}

	type loginReq struct {
		CreateAccount         bool `json:"CreateAccount"`
		EncryptedRequest      any  `json:"EncryptedRequest"`
		InfoRequestParameters struct {
			GetPlayerProfile bool `json:"GetPlayerProfile"`
			GetUserAccountInfo bool `json:"GetUserAccountInfo"`
		} `json:"InfoRequestParameters"`
		PlayerSecret any    `json:"PlayerSecret"`
		TitleID      string `json:"TitleId"`
		XboxToken    string `json:"XboxToken"`
	}

	type loginResp struct {
		Code   int    `json:"code"`
		Status string `json:"status"`
		Data   struct {
			SessionTicket string `json:"SessionTicket"`
			PlayFabID     string `json:"PlayFabId"`
			EntityToken   struct {
				EntityToken string `json:"EntityToken"`
			} `json:"EntityToken"`
		} `json:"data"`
	}

	req := loginReq{
		CreateAccount: true,
		TitleID:       strings.ToUpper(playfabTitleID),
		XboxToken: fmt.Sprintf("XBL3.0 x=%s;%s",
			xbl.AuthorizationToken.DisplayClaims.UserInfo[0].UserHash,
			xbl.AuthorizationToken.Token),
	}
	req.InfoRequestParameters.GetPlayerProfile = true
	req.InfoRequestParameters.GetUserAccountInfo = true

	var resp loginResp
	if err := p.playfabRequest(
		fmt.Sprintf("Client/LoginWithXbox?sdk=%s", playfabSDK),
		req, &resp,
	); err != nil {
		return err
	}

	p.playFabID = resp.Data.PlayFabID
	p.session = resp.Data.SessionTicket
	p.entityToken = resp.Data.EntityToken.EntityToken
	return nil
}

func (p *PlayFabClient) getEntityToken() error {
	type entityReq struct {
		Entity struct {
			ID   string `json:"Id"`
			Type string `json:"Type"`
		} `json:"Entity"`
	}
	type entityResp struct {
		Code int `json:"code"`
		Data struct {
			EntityToken string `json:"EntityToken"`
		} `json:"data"`
	}

	req := entityReq{}
	req.Entity.ID = p.playFabID
	req.Entity.Type = "master_player_account"

	var resp entityResp
	if err := p.playfabRequest(
		fmt.Sprintf("Authentication/GetEntityToken?sdk=%s", playfabSDK),
		req, &resp,
	); err != nil {
		return err
	}
	p.entityToken = resp.Data.EntityToken
	return nil
}

func (p *PlayFabClient) getMCToken() error {
	type device struct {
		ApplicationType    string   `json:"applicationType"`
		Capabilities       []string `json:"capabilities"`
		GameVersion        string   `json:"gameVersion"`
		ID                 string   `json:"id"`
		Memory             string   `json:"memory"`
		Platform           string   `json:"platform"`
		PlayFabTitleID     string   `json:"playFabTitleId"`
		StorePlatform      string   `json:"storePlatform"`
		TreatmentOverrides any      `json:"treatmentOverrides"`
		Type               string   `json:"type"`
	}
	type user struct {
		Language     string `json:"language"`
		LanguageCode string `json:"languageCode"`
		RegionCode   string `json:"regionCode"`
		Token        string `json:"token"`
		TokenType    string `json:"tokenType"`
	}
	type mcReq struct {
		Device device `json:"device"`
		User   user   `json:"user"`
	}
	type mcResp struct {
		Result struct {
			AuthorizationHeader string `json:"authorizationHeader"`
		} `json:"result"`
	}

	dev := device{
		ApplicationType: "MinecraftPE",
		Capabilities:    []string{"RayTracing"},
		GameVersion:     protocol.CurrentVersion,
		ID:              uuid.New().String(),
		Memory:          fmt.Sprint(16 * 1024 * 1024 * 1024),
		Platform:        "Windows10",
		PlayFabTitleID:  strings.ToUpper(playfabTitleID),
		StorePlatform:   "uwp.store",
		Type:            "Windows10",
	}
	usr := user{
		Language:     "en",
		LanguageCode: "en-US",
		RegionCode:   "US",
		Token:        p.session,
		TokenType:    "PlayFab",
	}

	// First request without treatment overrides
	req := mcReq{Device: dev, User: usr}
	var resp mcResp
	if err := p.externalRequest(sessionStartURL, req, &resp); err != nil {
		return err
	}
	if resp.Result.AuthorizationHeader == "" {
		return fmt.Errorf("empty authorization header from first session/start")
	}

	// Second request with treatment overrides including signaling WebSockets
	req.Device.TreatmentOverrides = []string{
		"mc-signaling-usewebsockets",
		"mc-signaling-useturn",
	}
	resp = mcResp{}
	if err := p.externalRequest(sessionStartURL, req, &resp); err != nil {
		return err
	}
	if resp.Result.AuthorizationHeader == "" {
		return fmt.Errorf("empty authorization header from second session/start")
	}

	p.mcToken = resp.Result.AuthorizationHeader
	return nil
}
