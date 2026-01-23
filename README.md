# Burnodd Minecraft Server

Tools for managing a Minecraft Bedrock Realm with custom behavior packs and structure generation.

## Architecture

- **Behavior Pack**: Scripts that receive structure commands via scriptevent
- **Upload Tool**: Connects as player to send commands via gophertunnel
- **Generators**: Create 3D structure data (maze, sphere, cube, pyramid)

## Why This Works

The Realm upload tool authenticates with Xbox Live, then connects as a player via gophertunnel. The behavior pack's JavaScript API listens for scriptevent messages and builds structures block-by-block.

## Realm Setup

1. Create `.realm-invite` with your invite code
2. Run `make install-pack` to prepare the world with the behavior pack
3. Upload the generated `.mcworld` file to your Realm via Minecraft
4. Run `make upload` to send structures

## Quick Start

```bash
# One-time: Prepare the behavior pack for your Realm
make install-pack
# This downloads your Realm world, injects the pack, and saves to output/realm-with-pack.mcworld
# Then manually upload via Minecraft: Realm Settings > Replace World

# Generate a structure
make sphere RADIUS=10 BLOCK=minecraft:glass

# Upload to Realm
make upload
```

## Structure Generators

```bash
# Maze: WIDTH, HEIGHT, LENGTH, BLOCK
make maze WIDTH=15 HEIGHT=7 LENGTH=15 BLOCK=minecraft:stone_bricks

# Sphere: RADIUS, BLOCK, HOLLOW
make sphere RADIUS=5 BLOCK=minecraft:glass HOLLOW=true

# Cube: SIZE, BLOCK, HOLLOW
make cube SIZE=10 BLOCK=minecraft:stone HOLLOW=true

# Pyramid: BASE, BLOCK
make pyramid BASE=15 BLOCK=minecraft:sandstone

# Test patterns: PATTERN, SIZE
make test PATTERN=frame SIZE=10
```

## Custom Functions

Function files let you create custom commands.

### Creating a function

1. Create a `.mcfunction` file in:
   ```
   bedrock/behavior_packs/burnodd_scripts/functions/
   ```

2. Example `trap.mcfunction`:
   ```mcfunction
   # Surround the player with glass
   fill ~-2 ~ ~-2 ~2 ~3 ~2 glass hollow
   ```

3. Re-run `make install-pack` to update the Realm

4. Run with: `/function burnodd_scripts/<filename>`

### Included functions

- `fireworks` - Launches fireworks around the player
- `cube` - Creates a hollow glass cube

## Connecting

### Nintendo Switch
1. Open Minecraft and accept the Realm invite
2. Join from the Realms tab

### Windows/Steam Deck
1. Open Minecraft and accept the Realm invite
2. Join from the Realms tab

## Backups

World backups are automatically created before pack injection in `backups/`.
The modified world is saved to `output/realm-with-pack.mcworld`.

To skip backups:
```bash
tools/upload-realm/upload-realm -install-pack -no-backup
```

## Troubleshooting

### "behavior pack not found on Realm"
Run `make install-pack` to install the behavior pack to your Realm.

### Authentication issues
Delete `.realm-token` and re-run to authenticate again.

### Functions not working
- Verify behavior pack structure (manifest.json must be valid)
- Re-run `make install-pack` after adding new functions

## Files

```
minecraft-server/
├── Makefile                    # Build and upload commands
├── .realm-invite               # Your Realm invite code (create this)
├── bedrock/
│   └── behavior_packs/
│       └── burnodd_scripts/
│           ├── manifest.json   # Pack metadata
│           ├── scripts/main.js # Scriptevent handler
│           └── functions/      # Custom mcfunctions
├── tools/
│   ├── generators/             # Structure generators (JS)
│   └── upload-realm/           # Realm upload tool (Go)
└── backups/                    # World backups (auto-created)
```
