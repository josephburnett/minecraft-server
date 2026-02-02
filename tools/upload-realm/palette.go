package main

import (
	"fmt"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

// Palette holds mappings between block/item names and their network IDs,
// built from connection GameData and CreativeContent packets.
type Palette struct {
	// itemsByName maps item name (e.g. "minecraft:stone") to its network ID (int16).
	itemsByName map[string]int16
	// itemsByID maps item network ID back to name.
	itemsByID map[int16]string

	// creativeItems maps item name to the CreativeItem entry from the CreativeContent packet.
	// This is needed to get the CreativeItemNetworkID for ItemStackRequests.
	creativeItems map[string]protocol.CreativeItem
	// creativeItemsByBlockRuntimeID maps block runtime ID to creative item, for faster lookup.
	creativeItemsByBlockRuntimeID map[int32]protocol.CreativeItem
}

// NewPalette creates a Palette from the GameData items list.
// CreativeContent must be loaded separately via LoadCreativeContent.
func NewPalette(items []protocol.ItemEntry) *Palette {
	p := &Palette{
		itemsByName:                   make(map[string]int16),
		itemsByID:                     make(map[int16]string),
		creativeItems:                 make(map[string]protocol.CreativeItem),
		creativeItemsByBlockRuntimeID: make(map[int32]protocol.CreativeItem),
	}
	for _, item := range items {
		p.itemsByName[item.Name] = item.RuntimeID
		p.itemsByID[item.RuntimeID] = item.Name
	}
	return p
}

// LoadCreativeContent indexes the creative inventory items from a CreativeContent packet.
// This allows us to look up the CreativeItemNetworkID needed for CraftCreativeStackRequestAction.
func (p *Palette) LoadCreativeContent(pk *packet.CreativeContent) {
	for _, ci := range pk.Items {
		name, ok := p.itemsByID[int16(ci.Item.NetworkID)]
		if !ok {
			continue
		}
		// Store first occurrence per name (default state)
		if _, exists := p.creativeItems[name]; !exists {
			p.creativeItems[name] = ci
		}
		// Also index by block runtime ID if present
		if ci.Item.BlockRuntimeID != 0 {
			if _, exists := p.creativeItemsByBlockRuntimeID[ci.Item.BlockRuntimeID]; !exists {
				p.creativeItemsByBlockRuntimeID[ci.Item.BlockRuntimeID] = ci
			}
		}
	}
}

// ItemNetworkID returns the network ID for a block/item name.
func (p *Palette) ItemNetworkID(name string) (int16, bool) {
	id, ok := p.itemsByName[name]
	return id, ok
}

// CreativeItem returns the creative inventory entry for a block/item name.
func (p *Palette) CreativeItem(name string) (protocol.CreativeItem, bool) {
	ci, ok := p.creativeItems[name]
	return ci, ok
}

// ItemName returns the name for a network ID.
func (p *Palette) ItemName(id int16) (string, bool) {
	name, ok := p.itemsByID[id]
	return name, ok
}

// DumpStats prints palette statistics for debugging.
func (p *Palette) DumpStats() {
	fmt.Printf("Palette: %d items, %d creative items, %d creative items by block runtime ID\n",
		len(p.itemsByName), len(p.creativeItems), len(p.creativeItemsByBlockRuntimeID))

	// Print a few example mappings
	count := 0
	for name, ci := range p.creativeItems {
		if strings.HasPrefix(name, "minecraft:") && ci.Item.BlockRuntimeID != 0 {
			fmt.Printf("  %s -> itemNetID=%d blockRuntimeID=%d creativeNetID=%d\n",
				name, ci.Item.NetworkID, ci.Item.BlockRuntimeID, ci.CreativeItemNetworkID)
			count++
			if count >= 10 {
				break
			}
		}
	}
}
