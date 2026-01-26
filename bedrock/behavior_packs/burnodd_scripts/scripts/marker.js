import { world, system } from "@minecraft/server";

// Stored marker state - persists while world is loaded
let currentMarker = null;

/**
 * Get the current marker
 * @returns {{x: number, y: number, z: number, rotation: number, dimension: Dimension}|null}
 */
export function getMarker() {
    return currentMarker;
}

/**
 * Clear the current marker (state only, not blocks)
 */
export function clearMarker() {
    currentMarker = null;
}

/**
 * Consume the marker - removes blocks, clears state, returns position info
 * Call this BEFORE building to ensure marker is fully removed
 * @returns {{x: number, y: number, z: number, rotation: number, dimension: Dimension}|null}
 */
export function consumeMarker() {
    if (!currentMarker) {
        return null;
    }

    const marker = currentMarker;
    const { x, y, z, dimension } = marker;

    // Remove marker blocks FIRST
    removeMarkerBlocks(dimension, x, y, z);

    // Clear state
    currentMarker = null;

    return marker;
}

/**
 * Get player's facing as rotation (0, 90, 180, 270)
 * @param {Player} player
 * @returns {number}
 */
function getFacingRotation(player) {
    const rotation = player.getRotation();
    const yaw = rotation.y;
    const normalizedYaw = ((yaw % 360) + 360) % 360;

    if (normalizedYaw >= 315 || normalizedYaw < 45) {
        return 0;     // South (+Z)
    } else if (normalizedYaw >= 45 && normalizedYaw < 135) {
        return 270;   // West (-X)
    } else if (normalizedYaw >= 135 && normalizedYaw < 225) {
        return 180;   // North (-Z)
    } else {
        return 90;    // East (+X)
    }
}

/**
 * Place marker blocks based on rotation
 * @param {Dimension} dimension
 * @param {number} x
 * @param {number} y
 * @param {number} z
 * @param {number} rotation
 */
function placeMarkerBlocks(dimension, x, y, z, rotation) {
    // Place white origin block
    try {
        const origin = dimension.getBlock({ x, y, z });
        if (origin) origin.setType("minecraft:white_concrete");
    } catch (e) {}

    // Green Y-axis (always up)
    for (let i = 1; i < 5; i++) {
        try {
            const block = dimension.getBlock({ x, y: y + i, z });
            if (block) block.setType("minecraft:lime_concrete");
        } catch (e) {}
    }

    // Red and Blue arms based on rotation
    // rotation 0: red=+X, blue=+Z (south)
    // rotation 90: red=+Z, blue=-X (east)
    // rotation 180: red=-X, blue=-Z (north)
    // rotation 270: red=-Z, blue=+X (west)

    let redDx = 0, redDz = 0, blueDx = 0, blueDz = 0;
    switch (rotation) {
        case 0:   redDx = 1;  blueDz = 1;  break;
        case 90:  redDz = 1;  blueDx = -1; break;
        case 180: redDx = -1; blueDz = -1; break;
        case 270: redDz = -1; blueDx = 1;  break;
    }

    for (let i = 1; i < 5; i++) {
        try {
            const redBlock = dimension.getBlock({ x: x + redDx * i, y, z: z + redDz * i });
            if (redBlock) redBlock.setType("minecraft:red_concrete");
        } catch (e) {}
        try {
            const blueBlock = dimension.getBlock({ x: x + blueDx * i, y, z: z + blueDz * i });
            if (blueBlock) blueBlock.setType("minecraft:blue_concrete");
        } catch (e) {}
    }
}

/**
 * Remove marker blocks at a location
 * @param {Dimension} dimension
 * @param {number} x
 * @param {number} y
 * @param {number} z
 */
function removeMarkerBlocks(dimension, x, y, z) {
    const markerBlocks = ["minecraft:white_concrete", "minecraft:red_concrete", "minecraft:lime_concrete", "minecraft:blue_concrete"];

    // Check all possible marker block positions (covers all rotations)
    const positions = [
        { x, y, z }, // origin
    ];

    // Y axis
    for (let i = 1; i < 5; i++) {
        positions.push({ x, y: y + i, z });
    }

    // All horizontal directions (covers all rotations)
    for (let i = 1; i < 5; i++) {
        positions.push({ x: x + i, y, z });
        positions.push({ x: x - i, y, z });
        positions.push({ x, y, z: z + i });
        positions.push({ x, y, z: z - i });
    }

    for (const pos of positions) {
        try {
            const block = dimension.getBlock(pos);
            if (block && markerBlocks.includes(block.typeId)) {
                block.setType("minecraft:air");
            }
        } catch (e) {}
    }
}

/**
 * Place a marker at the player's position
 * @param {Player} player
 */
export function placeMarker(player) {
    const pos = player.location;
    const x = Math.floor(pos.x);
    const y = Math.floor(pos.y);
    const z = Math.floor(pos.z);
    const rotation = getFacingRotation(player);
    const dimension = player.dimension;

    // Remove old marker if exists
    if (currentMarker) {
        removeMarkerBlocks(currentMarker.dimension, currentMarker.x, currentMarker.y, currentMarker.z);
    }

    // Store new marker
    currentMarker = { x, y, z, rotation, dimension };

    // Place marker blocks
    placeMarkerBlocks(dimension, x, y, z, rotation);

    const dirs = ["South (+Z)", "East (+X)", "North (-Z)", "West (-X)"];
    const dirIndex = rotation / 90;
    player.sendMessage(`§aMarker placed at ${x}, ${y}, ${z} facing ${dirs[dirIndex]}`);
    player.sendMessage(`§7Red arm = forward direction`);
}

/**
 * Rotate the current marker 90° clockwise
 * @param {Player} player
 */
export function rotateMarker(player) {
    if (!currentMarker) {
        player.sendMessage("§cNo marker placed. Use /function burnodd_scripts/marker first.");
        return;
    }

    const { x, y, z, dimension } = currentMarker;

    // Remove old blocks
    removeMarkerBlocks(dimension, x, y, z);

    // Rotate 90° clockwise
    currentMarker.rotation = (currentMarker.rotation + 90) % 360;

    // Place new blocks
    placeMarkerBlocks(dimension, x, y, z, currentMarker.rotation);

    const dirs = ["South (+Z)", "East (+X)", "North (-Z)", "West (-X)"];
    const dirIndex = currentMarker.rotation / 90;
    player.sendMessage(`§aMarker rotated to face ${dirs[dirIndex]}`);
}

/**
 * Clear the current marker
 * @param {Player} player
 */
export function clearMarkerCommand(player) {
    if (!currentMarker) {
        player.sendMessage("§cNo marker to clear.");
        return;
    }

    const { x, y, z, dimension } = currentMarker;
    removeMarkerBlocks(dimension, x, y, z);
    currentMarker = null;

    player.sendMessage("§aMarker cleared.");
}

/**
 * Initialize marker scriptevent handlers
 */
export function initMarkerHandler() {
    system.afterEvents.scriptEventReceive.subscribe((event) => {
        const player = event.sourceEntity;
        if (!player) {
            world.sendMessage("§cMarker commands require a player source");
            return;
        }

        switch (event.id) {
            case "burnodd:marker":
                placeMarker(player);
                break;
            case "burnodd:marker_rotate":
                rotateMarker(player);
                break;
            case "burnodd:marker_clear":
                clearMarkerCommand(player);
                break;
        }
    });
}
