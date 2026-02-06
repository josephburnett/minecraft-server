import { world, system } from "@minecraft/server";
import { buildBlocksAt } from "./structure-builder.js";
import { consumeMarker } from "./marker.js";
import { generateCubeGrid, gridToBlocks, getPlayerRotation } from "./shapes.js";

/**
 * Build a cube at the marker location (if set) or player's location
 * @param {Player} player
 * @param {object} options
 * @param {number} [options.size=10] - Cube size
 * @param {string} [options.block="minecraft:stone"] - Block type
 * @param {boolean} [options.hollow=false] - If true, only surface blocks
 */
export function buildCube(player, options = {}) {
    const size = options.size || 10;
    const blockType = options.block || "minecraft:stone";
    const hollow = options.hollow || false;

    // Try to consume marker FIRST (removes blocks before building)
    const marker = consumeMarker();
    let buildX, buildY, buildZ, dimension;

    if (marker) {
        buildX = marker.x;
        buildY = marker.y;
        buildZ = marker.z;
        dimension = marker.dimension;
        player.sendMessage(`§aBuilding ${hollow ? "hollow " : ""}cube (${size}x${size}x${size}) at marker...`);
    } else {
        // Fall back to player position
        const pos = player.location;
        buildX = Math.floor(pos.x);
        buildY = Math.floor(pos.y);
        buildZ = Math.floor(pos.z);
        dimension = player.dimension;
        player.sendMessage(`§aBuilding ${hollow ? "hollow " : ""}cube (${size}x${size}x${size}) at player...`);
    }

    // Generate the cube
    const { grid, center } = generateCubeGrid(size, hollow);

    // Convert to blocks
    const blocks = gridToBlocks(grid, blockType);

    const totalBlocks = blocks.length;
    player.sendMessage(`§7Placing ${totalBlocks} blocks...`);

    // Build at the target location, offset so center is at build position
    buildBlocksAt(dimension, buildX - center, buildY - center, buildZ - center, blocks, (placed) => {
        player.sendMessage(`§a§lCube complete! §r§7${placed} blocks placed`);
    });
}

/**
 * Parse cube options from a message string
 * Format: "size block hollow" (all optional)
 * @param {string} message
 * @returns {object}
 */
export function parseCubeOptions(message) {
    const options = {};
    const parts = message.trim().split(/\s+/);

    if (parts[0] && !isNaN(parseInt(parts[0]))) {
        options.size = parseInt(parts[0]);
    }
    if (parts[1] && parts[1] !== "hollow" && parts[1] !== "solid") {
        options.block = parts[1].includes(":") ? parts[1] : `minecraft:${parts[1]}`;
    }
    if (parts.includes("hollow") || parts[2] === "true") {
        options.hollow = true;
    }

    return options;
}

/**
 * Initialize cube scriptevent handler
 */
export function initCubeHandler() {
    system.afterEvents.scriptEventReceive.subscribe((event) => {
        if (event.id === "burnodd:cube") {
            const player = event.sourceEntity;
            if (!player) {
                world.sendMessage("§cCube generation requires a player source");
                return;
            }

            try {
                const options = parseCubeOptions(event.message || "");
                buildCube(player, options);
            } catch (e) {
                player.sendMessage(`§cCube generation failed: ${e.message}`);
            }
        }
    });
}
