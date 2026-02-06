import { world, system } from "@minecraft/server";
import { ABILITIES } from "./abilities.js";
import { hasPermission } from "./permissions.js";
import { initChunkReceiver } from "./chunk-receiver.js";
import { initMazeHandler } from "./maze.js";
import { initMarkerHandler } from "./marker.js";
import { initSphereHandler } from "./sphere.js";
import { initCubeHandler } from "./cube.js";

// =============================================================================
// BLOCK PROTECTION
// =============================================================================
world.beforeEvents.playerBreakBlock.subscribe((event) => {
    if (event.block.typeId === "minecraft:obsidian") {
        event.cancel = true;
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
// INITIALIZATION
// =============================================================================
initChunkReceiver();
initMazeHandler();
initMarkerHandler();
initSphereHandler();
initCubeHandler();

// =============================================================================
// CONNECTION DIAGNOSTICS
// =============================================================================
world.afterEvents.playerJoin.subscribe((event) => {
    world.sendMessage(`§8[diag] playerJoin: ${event.playerName}`);
});

world.afterEvents.playerSpawn.subscribe((event) => {
    world.sendMessage(`§8[diag] playerSpawn: ${event.player.name} initialSpawn=${event.initialSpawn}`);
});

world.afterEvents.playerLeave.subscribe((event) => {
    world.sendMessage(`§8[diag] playerLeave: ${event.playerName}`);
});

world.beforeEvents.chatSend.subscribe((event) => {
    world.sendMessage(`§8[diag] chatSend: "${event.message}" from ${event.sender.name}`);
});

system.afterEvents.scriptEventReceive.subscribe((event) => {
    world.sendMessage(`§8[diag] scriptEvent: id=${event.id} msg="${event.message}" src=${event.sourceType}`);
});

// =============================================================================
// STARTUP
// =============================================================================
world.afterEvents.worldLoad.subscribe(() => {
    const enabled = Object.entries(ABILITIES).filter(([_, a]) => a.permission !== "disabled");
    world.sendMessage(`§aBurnodd scripts loaded! §7(${enabled.length} abilities active)`);
    world.sendMessage(`§7Commands: marker, maze, sphere, cube`);
});
