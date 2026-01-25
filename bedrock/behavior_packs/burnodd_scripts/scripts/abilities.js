import { system } from "@minecraft/server";

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
            }, 1);
        }
    }
};
