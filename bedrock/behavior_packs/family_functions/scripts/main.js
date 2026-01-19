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

    // Emerald - creates 10x10x10 hollow stone cube
    "minecraft:emerald": {
        name: "Cube Builder",
        permission: "operator",
        action: (player, blockHit) => {
            const pos = blockHit.block.location;
            player.runCommand(`fill ${pos.x} ${pos.y} ${pos.z} ${pos.x + 9} ${pos.y + 9} ${pos.z + 9} stone hollow`);
            player.sendMessage(`§aCube created at: ${pos.x}, ${pos.y}, ${pos.z}`);
        }
    },

    // Fire Charge - NUKE! destroys 10 block radius sphere
    "minecraft:fire_charge": {
        name: "Nuke",
        permission: "operator",
        maxDistance: 100,
        action: (player, blockHit) => {
            const center = blockHit.block.location;
            const radius = 10;
            const dimension = player.dimension;
            let destroyed = 0;

            for (let x = -radius; x <= radius; x++) {
                for (let y = -radius; y <= radius; y++) {
                    for (let z = -radius; z <= radius; z++) {
                        if (x*x + y*y + z*z <= radius*radius) {
                            try {
                                const block = dimension.getBlock({
                                    x: center.x + x,
                                    y: center.y + y,
                                    z: center.z + z
                                });
                                if (block && block.typeId !== "minecraft:air" && block.typeId !== "minecraft:bedrock") {
                                    block.setType("minecraft:air");
                                    destroyed++;
                                }
                            } catch (e) {}
                        }
                    }
                }
            }
            player.sendMessage(`§c§lNUKED! §r§7Destroyed ${destroyed} blocks`);
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
