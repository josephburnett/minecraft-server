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

    // Emerald - creates 10x10x10 hollow stone cube centered on player
    "minecraft:emerald": {
        name: "Cube Builder",
        permission: "operator",
        action: (player, blockHit) => {
            const pos = player.location;
            const x = Math.floor(pos.x);
            const y = Math.floor(pos.y);
            const z = Math.floor(pos.z);
            player.runCommand(`fill ${x - 5} ${y - 5} ${z - 5} ${x + 4} ${y + 4} ${z + 4} stone hollow`);
            player.sendMessage(`§aCube created around you`);
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
});
