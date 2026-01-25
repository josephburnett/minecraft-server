import { world } from "@minecraft/server";
import { ABILITIES } from "./abilities.js";
import { hasPermission } from "./permissions.js";
import { initChunkReceiver } from "./chunk-receiver.js";
import { initMazeHandler } from "./maze.js";

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

// =============================================================================
// STARTUP
// =============================================================================
world.afterEvents.worldLoad.subscribe(() => {
    const enabled = Object.entries(ABILITIES).filter(([_, a]) => a.permission !== "disabled");
    world.sendMessage(`§aBurnodd scripts loaded! §7(${enabled.length} abilities active)`);
    world.sendMessage(`§7Commands: /function burnodd_scripts/maze`);
});
