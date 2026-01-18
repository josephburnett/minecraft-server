# Family Minecraft Server

Bedrock Dedicated Server with BedrockConnect for Nintendo Switch access.

## Architecture

```
┌──────────────────────────────────────────────────────────────────┐
│                        rtb (192.168.88.10)                       │
│                                                                  │
│  ┌──────────────┐  ┌───────────────────┐  ┌───────────────────┐  │
│  │     DNS      │  │   BedrockConnect  │  │       BDS         │  │
│  │   (BIND9)    │  │   (server list)   │  │  (game server)    │  │
│  │    :53       │  │    :19132/udp     │  │   :19133/udp      │  │
│  └──────────────┘  └───────────────────┘  └───────────────────┘  │
└──────────────────────────────────────────────────────────────────┘
         │                     │                      │
         │                     │                      │
    Switch (DNS)          Switch (via             Laptop/Steam Deck
    redirects to          BedrockConnect)         (direct connect)
    BedrockConnect        sees "Family Server"    192.168.88.10:19133
```

## How It Works

1. **Switch** has its DNS pointed to rtb (192.168.88.10)
2. When Switch tries to connect to a Featured Server (Lifeboat, Hive, etc.), DNS redirects to BedrockConnect
3. BedrockConnect shows a server list with only "Family Server"
4. Clicking "Family Server" connects to BDS on port 19133
5. **Laptop/Steam Deck** connect directly to 192.168.88.10:19133

## Server Setup (on rtb)

```bash
# Clone the repo
git clone <repo-url> ~/minecraft-server
cd ~/minecraft-server

# Start everything
docker compose up -d

# Check logs
docker compose logs -f

# Stop everything
docker compose down
```

## Switch Setup (one-time per Switch)

1. **Settings** → **Internet** → **Internet Settings**
2. Select your WiFi network → **Change Settings**
3. **DNS Settings** → **Manual**
4. **Primary DNS**: `192.168.88.10`
5. **Secondary DNS**: `8.8.8.8`
6. **Save**

## Connecting

### Nintendo Switch
1. Open Minecraft → **Play** → **Servers** tab
2. Click any Featured Server (Lifeboat, Hive, etc.)
3. BedrockConnect loads → shows "Family Server"
4. Click **Family Server** → you're in!

### Steam Deck (Bedrock Launcher)
1. Open Minecraft → **Play** → **Servers** tab
2. Click **Add Server**
3. Server Address: `192.168.88.10`
4. Port: `19133`
5. Save and connect

### Windows/Laptop (Bedrock Edition)
1. Open Minecraft → **Play** → **Servers** tab
2. Scroll down → **Add Server**
3. Server Address: `192.168.88.10`
4. Port: `19133`
5. Save and connect

## Custom Functions

Function files let you create custom commands the kids can run.

### Creating a function

1. Create a `.mcfunction` file in:
   ```
   bedrock/behavior_packs/family_functions/functions/
   ```

2. Example `trap.mcfunction`:
   ```mcfunction
   # Surround the player with glass
   fill ~-2 ~ ~-2 ~2 ~3 ~2 glass hollow
   ```

3. Restart the server OR run `/reload` in-game (requires cheats enabled)

4. Run with: `/function family_functions/<filename>`
   - Example: `/function family_functions/fireworks`

### Included functions

- `fireworks` - Launches fireworks around the player

## Server Console

To run server commands or see live output:

```bash
# Attach to console
docker attach minecraft-bedrock

# Detach without stopping: Ctrl+P, Ctrl+Q
```

Useful commands:
- `/reload` - Reload function files
- `/list` - Show connected players
- `/say <message>` - Broadcast message
- `/op <player>` - Give operator permissions

## Backups

World data is stored in a Docker volume (`bedrock-worlds`).

### Manual backup
```bash
# Find volume location
docker volume inspect minecraft-server_bedrock-worlds

# Or copy from container
docker cp minecraft-bedrock:/data/worlds ./backup-$(date +%Y%m%d)
```

### Restore
```bash
docker cp ./backup-folder/. minecraft-bedrock:/data/worlds/
docker restart minecraft-bedrock
```

## Troubleshooting

### Switch connects to real Featured Server instead of BedrockConnect
- Verify DNS settings on Switch (Primary: 192.168.88.10)
- Check DNS container is running: `docker compose ps`
- Try a different Featured Server (Lifeboat works well)

### "Unable to connect to world"
- Check BDS is running: `docker compose logs bedrock`
- Verify port 19133/udp is accessible
- Check firewall on rtb

### Functions not working
- Verify behavior pack structure (manifest.json must be valid)
- Run `/reload` after adding new functions
- Check BDS logs for errors loading packs

### BedrockConnect shows empty list
- Check `bedrockconnect/serverlist.json` has correct IP
- Restart bedrockconnect: `docker compose restart bedrockconnect`

## Ports Used

| Service        | Port      | Protocol |
|----------------|-----------|----------|
| DNS            | 53        | UDP/TCP  |
| BedrockConnect | 19132     | UDP      |
| BDS (game)     | 19133     | UDP      |

## Files

```
minecraft-server/
├── docker-compose.yml          # Container definitions
├── bedrockconnect/
│   └── serverlist.json         # Server list shown to Switch
├── bedrock/
│   └── behavior_packs/
│       └── family_functions/
│           ├── manifest.json   # Pack metadata
│           └── functions/
│               └── fireworks.mcfunction
├── dns/
│   ├── named.conf.options      # BIND9 options
│   ├── named.conf.local        # Zone definitions
│   └── db.minecraft            # DNS records (points to rtb)
└── README.md
```
