import { world, system } from "@minecraft/server";

// =============================================================================
// ABILITY CONFIGURATION
// Permission levels: "disabled" | "operator" | "everyone"
// =============================================================================
const ABILITIES = {
    // Stick - shows block info + places glowstone
    "minecraft:stick": {
        name: "Block Inspector",
        permission: "operator",
        action: (player, blockHit) => {
            const block = blockHit.block;
            const pos = block.location;
            player.sendMessage(`§eLooking at: §f${block.typeId} §7at ${pos.x}, ${pos.y}, ${pos.z}`);

            const face = blockHit.faceLocation;
            player.runCommand(`setblock ${Math.floor(face.x)} ${Math.floor(face.y)} ${Math.floor(face.z)} glowstone`);
        }
    },

    // Blaze Rod - destroys single block
    "minecraft:blaze_rod": {
        name: "Block Destroyer",
        permission: "operator",
        action: (player, blockHit) => {
            const block = blockHit.block;
            const pos = block.location;
            player.sendMessage(`§cDestroying: §f${block.typeId}`);
            player.runCommand(`setblock ${pos.x} ${pos.y} ${pos.z} air destroy`);
        }
    },

    // Diamond - marks location with diamond block
    "minecraft:diamond": {
        name: "Location Marker",
        permission: "operator",
        action: (player, blockHit) => {
            const pos = blockHit.block.location;
            player.runCommand(`setblock ${pos.x} ${pos.y} ${pos.z} diamond_block`);
            player.sendMessage(`§bMarked location: ${pos.x}, ${pos.y}, ${pos.z}`);
        }
    },

    // Snowball - spawns 100 bunnies
    "minecraft:snowball": {
        name: "Bunny Bomb",
        permission: "operator",
        action: (player, blockHit) => {
            const pos = blockHit.block.location;
            for (let i = 0; i < 100; i++) {
                player.runCommand(`summon rabbit ${pos.x} ${pos.y + 1} ${pos.z}`);
            }
            player.sendMessage(`§d§l100 BUNNIES! §r§7at ${pos.x}, ${pos.y}, ${pos.z}`);
        }
    },

    // Rabbit Foot - kills all rabbits
    "minecraft:rabbit_foot": {
        name: "Bunny Apocalypse",
        permission: "operator",
        action: (player, blockHit) => {
            player.runCommand(`kill @e[type=rabbit]`);
            player.sendMessage(`§c§lBUNNIES ELIMINATED!`);
        }
    },

    // Fire Charge - NUKE! destroys blocks in sphere (async to prevent hang)
    "minecraft:fire_charge": {
        name: "Nuke",
        permission: "operator",
        maxDistance: 100,
        action: (player, blockHit) => {
            const center = blockHit.block.location;
            const radius = 20;
            const dimension = player.dimension;
            const blocksPerTick = 5000;

            player.sendMessage(`§c§lNUKE INITIATED! §r§7Radius: ${radius} blocks...`);

            // State for incremental processing
            let currentX = -radius;
            let currentY = -radius;
            let currentZ = -radius;
            let destroyed = 0;
            let done = false;

            const intervalId = system.runInterval(() => {
                let processed = 0;

                while (!done && processed < blocksPerTick) {
                    if (currentX*currentX + currentY*currentY + currentZ*currentZ <= radius*radius) {
                        try {
                            const block = dimension.getBlock({
                                x: center.x + currentX,
                                y: center.y + currentY,
                                z: center.z + currentZ
                            });
                            if (block && block.typeId !== "minecraft:air" && block.typeId !== "minecraft:bedrock") {
                                block.setType("minecraft:air");
                                destroyed++;
                            }
                        } catch (e) {}
                    }
                    processed++;

                    // Advance position
                    currentZ++;
                    if (currentZ > radius) {
                        currentZ = -radius;
                        currentY++;
                        if (currentY > radius) {
                            currentY = -radius;
                            currentX++;
                            if (currentX > radius) {
                                done = true;
                            }
                        }
                    }
                }

                if (done) {
                    system.clearRun(intervalId);
                    player.sendMessage(`§c§lNUKE COMPLETE! §r§7Destroyed ${destroyed} blocks`);
                }
            }, 1); // Run every tick
        }
    }
};

// =============================================================================
// PERMISSION HELPERS
// =============================================================================
function isOperator(player) {
    // Check if player can run op-level commands
    try {
        // This command only succeeds for operators
        player.runCommand("testfor @s[tag=__op_check__]");
        return true;
    } catch (e) {
        // If it fails, check if it's a permission error or just no match
        // Operators can run the command (even if no match), non-ops get permission error
        return !e.message?.includes("permission");
    }
}

function hasPermission(player, ability) {
    if (ability.permission === "disabled") return false;
    if (ability.permission === "everyone") return true;
    if (ability.permission === "operator") return isOperator(player);
    return false;
}

// =============================================================================
// STRUCTURE BUILDER
// =============================================================================

/**
 * Base64 decode (Bedrock JS doesn't have atob)
 */
function base64Decode(str) {
    const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
    let result = [];
    let buffer = 0;
    let bits = 0;

    for (let i = 0; i < str.length; i++) {
        const c = str[i];
        if (c === "=") break;
        const idx = chars.indexOf(c);
        if (idx === -1) continue;

        buffer = (buffer << 6) | idx;
        bits += 6;

        while (bits >= 8) {
            bits -= 8;
            result.push((buffer >> bits) & 0xff);
        }
    }

    return result;
}

/**
 * Decode bitfield format
 */
function decodeBitfield(data, size) {
    const bytes = base64Decode(data);
    const [width, height, length] = size;
    const blocks = [];

    let bitIndex = 0;
    for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
            for (let z = 0; z < length; z++) {
                const byteIndex = Math.floor(bitIndex / 8);
                const bitOffset = 7 - (bitIndex % 8);
                const bit = (bytes[byteIndex] >> bitOffset) & 1;

                if (bit === 1) {
                    blocks.push([x, y, z]);
                }
                bitIndex++;
            }
        }
    }

    return blocks;
}

/**
 * Decode palette format
 */
function decodePalette(data, size, palette) {
    const bytes = base64Decode(data);
    const [width, height, length] = size;
    const blocks = [];

    let index = 0;
    for (let x = 0; x < width; x++) {
        for (let y = 0; y < height; y++) {
            for (let z = 0; z < length; z++) {
                const paletteIndex = bytes[index];
                const blockType = palette[paletteIndex];

                if (blockType && blockType !== "minecraft:air") {
                    blocks.push([x, y, z, blockType]);
                }
                index++;
            }
        }
    }

    return blocks;
}

/**
 * Build a structure at player location
 */
function buildStructure(player, structure) {
    const dimension = player.dimension;
    const playerPos = player.location;
    const px = Math.floor(playerPos.x);
    const py = Math.floor(playerPos.y);
    const pz = Math.floor(playerPos.z);

    const origin = structure.origin || [0, 0, 0];
    let blocks = [];

    // Decode based on structure type
    if (structure.type === "bitfield") {
        const positions = decodeBitfield(structure.data, structure.size);
        const block = structure.block || "minecraft:stone";
        blocks = positions.map(([x, y, z]) => [x, y, z, block]);
    } else if (structure.type === "palette") {
        blocks = decodePalette(structure.data, structure.size, structure.palette);
    } else if (structure.type === "sparse") {
        blocks = structure.blocks || [];
    } else {
        player.sendMessage(`§cUnknown structure type: ${structure.type}`);
        return;
    }

    if (blocks.length === 0) {
        player.sendMessage(`§cNo blocks to place!`);
        return;
    }

    player.sendMessage(`§aBuilding structure: §f${blocks.length} blocks...`);

    // Async chunked building to avoid watchdog
    const blocksPerTick = 1000;
    let index = 0;
    let placed = 0;

    const intervalId = system.runInterval(() => {
        const endIndex = Math.min(index + blocksPerTick, blocks.length);

        for (; index < endIndex; index++) {
            const [sx, sy, sz, blockType] = blocks[index];

            // Calculate world position
            const wx = px + sx - origin[0];
            const wy = py + sy - origin[1];
            const wz = pz + sz - origin[2];

            try {
                const block = dimension.getBlock({ x: wx, y: wy, z: wz });
                if (block) {
                    block.setType(blockType);
                    placed++;
                }
            } catch (e) {
                // Block might be outside loaded chunks
            }
        }

        if (index >= blocks.length) {
            system.clearRun(intervalId);
            player.sendMessage(`§a§lBuild complete! §r§7Placed ${placed} blocks`);
        }
    }, 1);
}

/**
 * Get first online player
 */
function getFirstPlayer() {
    const players = world.getAllPlayers();
    return players.length > 0 ? players[0] : null;
}

// =============================================================================
// SCRIPTEVENT HANDLER
// =============================================================================
system.afterEvents.scriptEventReceive.subscribe((event) => {
    if (event.id === "family:build") {
        try {
            const json = base64Decode(event.message)
                .map(b => String.fromCharCode(b))
                .join("");
            const structure = JSON.parse(json);
            const player = getFirstPlayer();

            if (!player) {
                world.sendMessage("§cNo players online to build structure!");
                return;
            }

            buildStructure(player, structure);
        } catch (e) {
            world.sendMessage(`§cFailed to parse structure: ${e.message}`);
        }
    }
});

// =============================================================================
// MAIN EVENT HANDLER
// =============================================================================
world.afterEvents.itemUse.subscribe((event) => {
    const player = event.source;
    const item = event.itemStack;

    const ability = ABILITIES[item.typeId];
    if (!ability) return;

    if (!hasPermission(player, ability)) {
        return; // Silent - let vanilla behavior happen
    }

    const maxDistance = ability.maxDistance || 50;
    const blockHit = player.getBlockFromViewDirection({
        maxDistance: maxDistance,
        includeLiquidBlocks: false,
        includePassableBlocks: false
    });

    if (blockHit) {
        ability.action(player, blockHit);
    } else {
        player.sendMessage(`§7No block in range (${maxDistance} blocks)`);
    }
});

// =============================================================================
// STARTUP
// =============================================================================
world.afterEvents.worldLoad.subscribe(() => {
    const enabled = Object.entries(ABILITIES).filter(([_, a]) => a.permission !== "disabled");
    world.sendMessage(`§aFamily scripts loaded! §7(${enabled.length} abilities active)`);
    world.sendMessage(`§7Structure builder ready (scriptevent family:build)`);
});
