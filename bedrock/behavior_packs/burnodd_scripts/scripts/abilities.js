/**
 * Ability configuration
 * Permission levels: "disabled" | "operator" | "everyone"
 */
export const ABILITIES = {
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

    // Blaze Rod - small dig (radius 2 sphere, single tick)
    "minecraft:blaze_rod": {
        name: "Small Dig",
        permission: "operator",
        action: (player, blockHit) => {
            const center = blockHit.block.location;
            const radius = 2;
            const dimension = player.dimension;

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
                                }
                            } catch (e) {}
                        }
                    }
                }
            }
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

    // Fire Charge - big dig (radius 8 sphere, single tick)
    "minecraft:fire_charge": {
        name: "Big Dig",
        permission: "operator",
        maxDistance: 100,
        action: (player, blockHit) => {
            const center = blockHit.block.location;
            const radius = 8;
            const dimension = player.dimension;

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
                                }
                            } catch (e) {}
                        }
                    }
                }
            }
        }
    }
};
