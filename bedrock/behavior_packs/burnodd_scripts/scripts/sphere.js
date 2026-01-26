import { world, system } from "@minecraft/server";
import { buildBlocksAt } from "./structure-builder.js";
import { consumeMarker } from "./marker.js";
import { generateSphereGrid, gridToBlocks, getPlayerRotation } from "./shapes.js";

/**
 * Build a sphere at the marker location (if set) or player's location
 * @param {Player} player
 * @param {object} options
 * @param {number} [options.radius=5] - Sphere radius
 * @param {string} [options.block="minecraft:glass"] - Block type
 * @param {boolean} [options.hollow=false] - If true, only surface blocks
 */
export function buildSphere(player, options = {}) {
    const radius = options.radius || 5;
    const blockType = options.block || "minecraft:glass";
    const hollow = options.hollow || false;

    // Try to consume marker FIRST (removes blocks before building)
    const marker = consumeMarker();
    let buildX, buildY, buildZ, dimension;

    if (marker) {
        buildX = marker.x;
        buildY = marker.y;
        buildZ = marker.z;
        dimension = marker.dimension;
        player.sendMessage(`§aBuilding ${hollow ? "hollow " : ""}sphere (r=${radius}) at marker...`);
    } else {
        // Fall back to player position
        const pos = player.location;
        buildX = Math.floor(pos.x);
        buildY = Math.floor(pos.y);
        buildZ = Math.floor(pos.z);
        dimension = player.dimension;
        player.sendMessage(`§aBuilding ${hollow ? "hollow " : ""}sphere (r=${radius}) at player...`);
    }

    // Generate the sphere
    const { grid, size, center } = generateSphereGrid(radius, hollow);

    // Convert to blocks
    const blocks = gridToBlocks(grid, blockType);

    const totalBlocks = blocks.length;
    player.sendMessage(`§7Placing ${totalBlocks} blocks...`);

    // Build at the target location, offset so center is at build position
    buildBlocksAt(dimension, buildX - center, buildY - center, buildZ - center, blocks, (placed) => {
        player.sendMessage(`§a§lSphere complete! §r§7${placed} blocks placed`);
    });
}

/**
 * Parse sphere options from a message string
 * Format: "radius block hollow" (all optional)
 * @param {string} message
 * @returns {object}
 */
export function parseSphereOptions(message) {
    const options = {};
    const parts = message.trim().split(/\s+/);

    if (parts[0] && !isNaN(parseInt(parts[0]))) {
        options.radius = parseInt(parts[0]);
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
 * Initialize sphere scriptevent handler
 */
export function initSphereHandler() {
    system.afterEvents.scriptEventReceive.subscribe((event) => {
        if (event.id === "burnodd:sphere") {
            const player = event.sourceEntity;
            if (!player) {
                world.sendMessage("§cSphere generation requires a player source");
                return;
            }

            try {
                const options = parseSphereOptions(event.message || "");
                buildSphere(player, options);
            } catch (e) {
                player.sendMessage(`§cSphere generation failed: ${e.message}`);
            }
        }
    });
}
