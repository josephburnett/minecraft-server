package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	StatusStarting         = "starting"
	StatusWaitingForClient = "waiting_for_client"
	StatusConnectingToRealm = "connecting_to_realm"
	StatusConnected        = "connected"
	StatusDisconnected     = "disconnected"
)

const maxChatHistory = 100

// ChatMessage represents a single chat message with metadata.
type ChatMessage struct {
	Time      time.Time `json:"time"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
	Type      string    `json:"type"` // "incoming" or "outgoing"
}

// PlayerInfo represents an online player.
type PlayerInfo struct {
	Username string `json:"username"`
	XUID     string `json:"xuid"`
}

// EntityInfo represents a tracked nearby entity.
type EntityInfo struct {
	RuntimeID uint64    `json:"runtime_id"`
	Type      string    `json:"type"` // entity identifier or player name
	Position  mgl32.Vec3 `json:"position"`
}

// InventorySlot represents a single inventory slot.
type InventorySlot struct {
	Slot  int    `json:"slot"`
	Item  string `json:"item"`
	Count int    `json:"count"`
}

// GameState holds the thread-safe cached game state updated from intercepted packets.
type GameState struct {
	mu sync.RWMutex

	status string

	// Connections (set during proxy connect, cleared on disconnect)
	serverConn *minecraft.Conn
	clientConn *minecraft.Conn

	// Player identity (set on connect)
	displayName string
	xuid        string
	entityID    uint64 // our entity runtime ID

	// Position and rotation
	posX, posY, posZ float32
	pitch, yaw       float32
	dimension        int32

	// Inventory: map of window ID -> slots
	inventory map[byte][]protocol.ItemInstance

	// Chat history (ring buffer)
	chatHistory []ChatMessage

	// Online players
	players map[string]PlayerInfo // keyed by XUID

	// World info
	worldName string
	worldTime int64
	gameMode  int32
	spawnPos  protocol.BlockPos

	// Player attributes
	health     float32
	attributes map[string]float32

	// Nearby entities
	entities map[uint64]EntityInfo

	// Item registry from StartGame (for resolving network IDs to names)
	itemRegistry map[int32]string // network ID -> item name
}

// NewGameState creates a new GameState with initial status.
func NewGameState() *GameState {
	return &GameState{
		status:       StatusStarting,
		inventory:    make(map[byte][]protocol.ItemInstance),
		players:      make(map[string]PlayerInfo),
		attributes:   make(map[string]float32),
		entities:     make(map[uint64]EntityInfo),
		itemRegistry: make(map[int32]string),
	}
}

// SetStatus updates the connection status.
func (gs *GameState) SetStatus(status string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.status = status
}

// Status returns the current connection status.
func (gs *GameState) Status() string {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.status
}

// SetConnections stores the server and client connections.
func (gs *GameState) SetConnections(server, client *minecraft.Conn) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.serverConn = server
	gs.clientConn = client
}

// ClearConnections removes stored connections.
func (gs *GameState) ClearConnections() {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.serverConn = nil
	gs.clientConn = nil
}

// ServerConn returns the server connection (nil if not connected).
func (gs *GameState) ServerConn() *minecraft.Conn {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.serverConn
}

// SetIdentity stores the player's identity info.
func (gs *GameState) SetIdentity(displayName, xuid string, entityID uint64) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.displayName = displayName
	gs.xuid = xuid
	gs.entityID = entityID
}

// Identity returns the player's display name and XUID.
func (gs *GameState) Identity() (string, string) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.displayName, gs.xuid
}

// EntityID returns our entity runtime ID.
func (gs *GameState) EntityID() uint64 {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.entityID
}

// UpdatePosition sets the current player position.
func (gs *GameState) UpdatePosition(x, y, z, pitch, yaw float32) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.posX = x
	gs.posY = y
	gs.posZ = z
	gs.pitch = pitch
	gs.yaw = yaw
}

// Position returns the current player position and rotation.
func (gs *GameState) Position() (x, y, z, pitch, yaw float32, dimension int32) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.posX, gs.posY, gs.posZ, gs.pitch, gs.yaw, gs.dimension
}

// SetDimension updates the current dimension.
func (gs *GameState) SetDimension(dim int32) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.dimension = dim
}

// SetInventory replaces the full inventory for a window.
func (gs *GameState) SetInventory(windowID byte, items []protocol.ItemInstance) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.inventory[windowID] = items
}

// UpdateInventorySlot updates a single inventory slot.
func (gs *GameState) UpdateInventorySlot(windowID byte, slot int, item protocol.ItemInstance) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if _, ok := gs.inventory[windowID]; !ok {
		gs.inventory[windowID] = make([]protocol.ItemInstance, slot+1)
	}
	for len(gs.inventory[windowID]) <= slot {
		gs.inventory[windowID] = append(gs.inventory[windowID], protocol.ItemInstance{})
	}
	gs.inventory[windowID][slot] = item
}

// Inventory returns the current inventory slots as a simplified list.
func (gs *GameState) Inventory() []InventorySlot {
	gs.mu.RLock()
	defer gs.mu.RUnlock()

	var result []InventorySlot
	for _, items := range gs.inventory {
		for i, item := range items {
			if item.Stack.Count == 0 {
				continue
			}
			name := gs.resolveItemName(item.Stack.NetworkID)
			result = append(result, InventorySlot{
				Slot:  i,
				Item:  name,
				Count: int(item.Stack.Count),
			})
		}
	}
	return result
}

// resolveItemName converts a network ID to an item name using the registry.
// Must be called with at least a read lock held.
func (gs *GameState) resolveItemName(networkID int32) string {
	if name, ok := gs.itemRegistry[networkID]; ok {
		return name
	}
	return fmt.Sprintf("unknown:%d", networkID)
}

// AppendChat adds a message to the chat history ring buffer.
func (gs *GameState) AppendChat(msg ChatMessage) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.chatHistory = append(gs.chatHistory, msg)
	if len(gs.chatHistory) > maxChatHistory {
		gs.chatHistory = gs.chatHistory[len(gs.chatHistory)-maxChatHistory:]
	}
}

// ChatHistory returns the last n chat messages (up to maxChatHistory).
func (gs *GameState) ChatHistory(n int) []ChatMessage {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	if n <= 0 || n > len(gs.chatHistory) {
		n = len(gs.chatHistory)
	}
	start := len(gs.chatHistory) - n
	result := make([]ChatMessage, n)
	copy(result, gs.chatHistory[start:])
	return result
}

// AddPlayer adds a player to the online player list.
func (gs *GameState) AddPlayer(xuid, username string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.players[xuid] = PlayerInfo{Username: username, XUID: xuid}
}

// RemovePlayer removes a player from the online player list.
func (gs *GameState) RemovePlayer(xuid string) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	delete(gs.players, xuid)
}

// Players returns the list of online players.
func (gs *GameState) Players() []PlayerInfo {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	result := make([]PlayerInfo, 0, len(gs.players))
	for _, p := range gs.players {
		result = append(result, p)
	}
	return result
}

// SetWorldTime updates the world time.
func (gs *GameState) SetWorldTime(t int64) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.worldTime = t
}

// SetHealth updates the player's health.
func (gs *GameState) SetHealth(h float32) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.health = h
	gs.attributes["health"] = h
}

// SetAttribute updates a named attribute.
func (gs *GameState) SetAttribute(name string, value float32) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.attributes[name] = value
	if name == "minecraft:health" {
		gs.health = value
	}
}

// InitFromGameData populates world info from the StartGame GameData.
func (gs *GameState) InitFromGameData(gd minecraft.GameData) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.worldName = gd.WorldName
	gs.gameMode = gd.PlayerGameMode
	gs.worldTime = gd.Time
	gs.dimension = gd.Dimension
	gs.spawnPos = gd.WorldSpawn
	gs.posX = gd.PlayerPosition.X()
	gs.posY = gd.PlayerPosition.Y()
	gs.posZ = gd.PlayerPosition.Z()
	gs.pitch = gd.Pitch
	gs.yaw = gd.Yaw
	gs.health = 20 // default

	// Build item registry: map network ID (RuntimeID from StartGame) to name.
	// ItemEntry.RuntimeID is int16, ItemStack.NetworkID is int32 — they correspond.
	for _, item := range gd.Items {
		gs.itemRegistry[int32(item.RuntimeID)] = item.Name
	}

}

// WorldInfo returns the cached world information.
func (gs *GameState) WorldInfo() (worldName string, worldTime int64, gameMode int32, health float32, spawnPos protocol.BlockPos) {
	gs.mu.RLock()
	defer gs.mu.RUnlock()
	return gs.worldName, gs.worldTime, gs.gameMode, gs.health, gs.spawnPos
}

// AddEntity adds or updates a tracked entity.
func (gs *GameState) AddEntity(runtimeID uint64, entityType string, pos mgl32.Vec3) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	gs.entities[runtimeID] = EntityInfo{
		RuntimeID: runtimeID,
		Type:      entityType,
		Position:  pos,
	}
}

// RemoveEntity removes a tracked entity.
func (gs *GameState) RemoveEntity(runtimeID uint64) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	delete(gs.entities, runtimeID)
}

// UpdateEntityPosition updates an entity's position.
func (gs *GameState) UpdateEntityPosition(runtimeID uint64, pos mgl32.Vec3) {
	gs.mu.Lock()
	defer gs.mu.Unlock()
	if e, ok := gs.entities[runtimeID]; ok {
		e.Position = pos
		gs.entities[runtimeID] = e
	}
}
