# Connecting to Minecraft Bedrock Realms with Go: Fixing the Disconnection Problem

Your gophertunnel client is likely disconnecting because it's **not maintaining a continuous packet read loop** after spawning. The Bedrock protocol requires clients to continuously process incoming packets and respond to keepalive messages—failure to do so triggers a ~30-second timeout disconnect. This report provides the complete technical solution.

## The core issue: missing game loop

When a Bedrock client connects and spawns, it must enter a continuous loop that reads and processes packets from the server. The server sends **NetworkStackLatencyPacket** (tick_sync) messages as keepalives—if the client doesn't respond within approximately 30 seconds, the server disconnects it. Simply calling `DoSpawn()` and then sending commands is insufficient; you must maintain an active packet processing loop for the entire session lifetime.

The minimum viable client structure in gophertunnel looks like this:

```go
conn, err := dialer.Dial("raknet", address)
if err != nil {
    return err
}
defer conn.Close()

if err := conn.DoSpawn(); err != nil {
    return err
}

// CRITICAL: Must continuously read packets to stay connected
for {
    pk, err := conn.ReadPacket()
    if err != nil {
        var disc minecraft.DisconnectError
        if errors.As(err, &disc) {
            log.Printf("Disconnected: %s", disc.Error())
        }
        return err
    }
    
    // Handle packets as needed (most are handled internally)
    switch p := pk.(type) {
    case *packet.Text:
        log.Printf("Chat: %s", p.Message)
    }
}
```

## Protocol requirements for maintaining Realm connections

The Minecraft Bedrock protocol uses RakNet (UDP-based) as its transport layer, with **AES-256-CFB8 encryption** mandatory for Realm connections. After the RakNet handshake completes, the client must progress through five distinct phases before entering normal gameplay.

**Login sequence that gophertunnel handles automatically:**

| Phase | Packets | Client Action |
|-------|---------|---------------|
| Network Settings | RequestNetworkSettings → NetworkSettings | Enable compression |
| Authentication | Login → PlayStatus(LOGIN_SUCCESS) | JWT chain with Xbox auth |
| Encryption | ServerToClientHandshake → ClientToServerHandshake | Enable AES encryption |
| Resource Packs | ResourcePacksInfo → ResourcePackClientResponse | Accept/decline packs |
| Spawn | StartGame, Chunks → PlayStatus(PLAYER_SPAWN) | Send SetLocalPlayerAsInitialised |

The `DoSpawn()` method handles all of this automatically—the problem occurs **after** spawn when the connection must be maintained.

## Three packets that cause disconnection if ignored

**NetworkStackLatencyPacket** functions as the primary keepalive mechanism. The server sends these periodically, and the client must echo them back. Gophertunnel handles this automatically, but only if you're calling `ReadPacket()` in a loop. Without continuous reading, the internal handlers never execute, and the timeout triggers after roughly 30 seconds.

**MovePlayerPacket** corrections from the server must be acknowledged in server-authoritative movement mode. If the client ignores position corrections, the server may disconnect for protocol violations.

**DisconnectPacket** indicates the server is terminating the connection. Always handle this gracefully rather than treating it as an unexpected error.

## Correct implementation pattern for Realm bots

Here's a complete working structure for a gophertunnel Realm client:

```go
package main

import (
    "context"
    "errors"
    "log"
    "time"

    "github.com/sandertv/gophertunnel/minecraft"
    "github.com/sandertv/gophertunnel/minecraft/auth"
    "github.com/sandertv/gophertunnel/minecraft/protocol/packet"
    "github.com/sandertv/gophertunnel/minecraft/realms"
)

func main() {
    // Get auth token source (interactive login first time, cached after)
    tokenSource := auth.TokenSource

    // Create Realms client and get address
    realmsClient := realms.NewClient(tokenSource)
    realm, err := realmsClient.Realm(context.Background(), "YOUR_INVITE_CODE")
    if err != nil {
        log.Fatal(err)
    }
    
    address, err := realm.Address(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    // Configure dialer with recommended settings
    dialer := minecraft.Dialer{
        TokenSource:       tokenSource,
        EnableClientCache: true, // Important for some servers
    }

    conn, err := dialer.Dial("raknet", address)
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    if err := conn.DoSpawn(); err != nil {
        log.Fatal(err)
    }

    log.Println("Spawned successfully, entering game loop")

    // Send a command after a short delay
    go func() {
        time.Sleep(2 * time.Second)
        cmd := &packet.CommandRequest{
            CommandLine: "scriptevent myaddon:test Hello World",
            CommandOrigin: protocol.CommandOrigin{
                Origin: protocol.CommandOriginPlayer,
            },
        }
        if err := conn.WritePacket(cmd); err != nil {
            log.Printf("Failed to send command: %v", err)
        }
    }()

    // CRITICAL: Game loop - must run continuously
    for {
        pk, err := conn.ReadPacket()
        if err != nil {
            var disc minecraft.DisconnectError
            if errors.As(err, &disc) {
                log.Printf("Server disconnected: %s", disc.Error())
            } else {
                log.Printf("Connection error: %v", err)
            }
            return
        }

        switch p := pk.(type) {
        case *packet.Text:
            log.Printf("[Chat] %s: %s", p.SourceName, p.Message)
        case *packet.CommandOutput:
            log.Printf("[Command Output] Success: %v, Messages: %v", 
                p.SuccessCount > 0, p.OutputMessages)
        case *packet.Disconnect:
            log.Printf("Disconnect packet: %s", p.Message)
            return
        }
    }
}
```

## Understanding scriptevent command requirements

The `/scriptevent` command has specific constraints that may cause silent failures. The namespace **cannot** be `minecraft:`—you must use a custom namespace like `myaddon:eventname`. The command requires that the world has either cheats enabled or the player has operator permissions. The format is:

```
/scriptevent <namespace:event_id> [message]
```

Use `CommandRequest` packet rather than `Text` packet for commands—the Text packet treats input as chat, while CommandRequest executes it as a direct command with proper permission checking.

## Debugging disconnections with ProxyPass

When the disconnect reason is unclear, intercept traffic using the **ProxyPass** tool (Kas-tle fork supports online-mode/Realms):

```bash
git clone https://github.com/Kas-tle/ProxyPass
cd ProxyPass
./gradlew shadowJar
java -jar build/libs/ProxyPass-*-all.jar
```

Configure `config.yml` to log all packets except high-frequency ones:

```yaml
log-packets: true
log-to: file
ignored-packets:
  - LevelChunkPacket
  - MovePlayerPacket
  - PlayerAuthInputPacket
  - NetworkStackLatencyPacket
```

This reveals exactly which packet exchange fails before disconnection.

## Alternative implementations that work with Realms

**bedrock-protocol (Node.js)** is the most mature library with explicit Realms support. If Go isn't mandatory, this provides a simpler path:

```javascript
const bedrock = require('bedrock-protocol')

const client = bedrock.createClient({
    realms: {
        pickRealm: (realms) => realms.find(r => r.name === 'My Realm')
    }
})

client.on('spawn', () => {
    console.log('Spawned!')
    client.queue('command_request', {
        command: 'scriptevent myaddon:test hello',
        origin: { type: 'player', uuid: '', request_id: '' },
        internal: false
    })
})

client.on('text', (packet) => {
    console.log(`Chat: ${packet.message}`)
})
```

This handles the game loop internally, making disconnection issues far less common.

## Common gophertunnel mistakes and solutions

**Problem: Context deadline exceeded during dial**
Solution: The Realm may be offline. The `realm.Address()` call blocks while the Realm starts—use a context with timeout and retry logic.

**Problem: Crash with "makeslice: len out of range"**
Solution: This was a RakNet-level bug in older gophertunnel versions. Update to v1.51.1 or later.

**Problem: 100% CPU usage, no chunks loading**
Solution: Set `EnableClientCache: true` in the Dialer configuration.

**Problem: Disconnection immediately after DoSpawn**
Solution: You're not entering a packet read loop. The connection must continuously process packets.

## Conclusion

The disconnection stems from not maintaining an active packet processing loop after spawning. Gophertunnel's `DoSpawn()` handles the complex login/spawn sequence automatically, but the client must then continuously call `ReadPacket()` to process keepalives and server messages. The library handles NetworkStackLatencyPacket responses internally, but only when packets are being read.

For Realm connections specifically, ensure you're using the Realms API (`minecraft/realms` package) to obtain the dynamic server address rather than hardcoding it, authenticate with a valid Xbox Live token source, and set `EnableClientCache: true` for compatibility. The command you're sending via scriptevent should work once the connection remains stable—verify the namespace isn't `minecraft:` and that the world allows commands.