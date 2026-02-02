package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func TestCommandRequestSerialization(t *testing.T) {
	testUUID := uuid.New()

	// With leading slash (as sent by real Bedrock clients)
	pk := &packet.CommandRequest{
		CommandLine: "/scriptevent burnodd:chunk test:0:1:dGVzdA==",
		CommandOrigin: protocol.CommandOrigin{
			Origin: protocol.CommandOriginPlayer,
			UUID:   testUUID,
		},
		Internal: false,
		Version:  "",
	}

	var buf bytes.Buffer

	// Write header like WritePacket does
	hdr := packet.Header{PacketID: pk.ID()}
	hdr.Write(&buf)

	// Write body
	w := protocol.NewWriter(&buf, 0)
	pk.Marshal(w)

	b := buf.Bytes()
	fmt.Printf("Total with header: %d bytes\n", len(b))
	fmt.Printf("UUID: %s\n", testUUID)
	fmt.Printf("Hex:\n")
	for i, v := range b {
		fmt.Printf("%02x ", v)
		if (i+1)%16 == 0 {
			fmt.Println()
		}
	}
	fmt.Println()

	// Annotate the byte layout
	offset := 0
	fmt.Printf("\nField layout:\n")
	fmt.Printf("  [%d] Header (varuint32): packet ID %d\n", offset, pk.ID())
	offset++ // header is 1 byte for ID 77

	cmdLen := len(pk.CommandLine)
	fmt.Printf("  [%d] CommandLine length (varuint32): %d\n", offset, cmdLen)
	offset++ // length is 1 byte for len < 128
	fmt.Printf("  [%d-%d] CommandLine: %q\n", offset, offset+cmdLen-1, pk.CommandLine)
	offset += cmdLen

	fmt.Printf("  [%d] Origin string length (varuint32)\n", offset)
	offset++
	fmt.Printf("  [%d-%d] Origin string: \"player\"\n", offset, offset+5)
	offset += 6

	fmt.Printf("  [%d-%d] UUID (16 bytes, swapped halves + reversed)\n", offset, offset+15)
	offset += 16

	fmt.Printf("  [%d] RequestID string length: 0\n", offset)
	offset++

	fmt.Printf("  [%d-%d] PlayerUniqueID (int64, little-endian): 0\n", offset, offset+7)
	offset += 8

	fmt.Printf("  [%d] Internal (bool): false\n", offset)
	offset++

	fmt.Printf("  [%d] Version string length: 0\n", offset)
	offset++

	fmt.Printf("  Total: %d bytes (expected %d)\n", len(b), offset)
}
