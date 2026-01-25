import { world, system } from "@minecraft/server";
import { base64Decode, buildStructure } from "./structure-builder.js";

// Storage for chunked transfers
const chunkSessions = {};

/**
 * Get first online player
 * @returns {Player|null}
 */
function getFirstPlayer() {
    const players = world.getAllPlayers();
    return players.length > 0 ? players[0] : null;
}

/**
 * Handle incoming chunk data
 * @param {string} message - Format: sessionId:chunkIndex:totalChunks:data
 */
function handleChunk(message) {
    const colonIdx1 = message.indexOf(":");
    const colonIdx2 = message.indexOf(":", colonIdx1 + 1);
    const colonIdx3 = message.indexOf(":", colonIdx2 + 1);

    const sessionId = message.substring(0, colonIdx1);
    const chunkIndex = parseInt(message.substring(colonIdx1 + 1, colonIdx2));
    const totalChunks = parseInt(message.substring(colonIdx2 + 1, colonIdx3));
    const data = message.substring(colonIdx3 + 1);

    // Initialize session if needed
    if (!chunkSessions[sessionId]) {
        chunkSessions[sessionId] = {
            total: totalChunks,
            chunks: [],
            received: 0
        };
        world.sendMessage(`§7Receiving structure (${totalChunks} chunks)...`);
    }

    const session = chunkSessions[sessionId];
    session.chunks[chunkIndex] = data;
    session.received++;

    // Check if all chunks received
    if (session.received === session.total) {
        const base64 = session.chunks.join("");
        delete chunkSessions[sessionId];

        const json = base64Decode(base64)
            .map(b => String.fromCharCode(b))
            .join("");
        const structure = JSON.parse(json);
        const player = getFirstPlayer();

        if (!player) {
            world.sendMessage("§cNo players online to build structure!");
            return;
        }

        buildStructure(player, structure);
    }
}

/**
 * Initialize the chunk receiver scriptevent handler
 */
export function initChunkReceiver() {
    system.afterEvents.scriptEventReceive.subscribe((event) => {
        if (event.id === "burnodd:chunk") {
            try {
                handleChunk(event.message);
            } catch (e) {
                world.sendMessage(`§cChunk transfer failed: ${e.message}`);
            }
        }
    });
}
