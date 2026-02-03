import { world, system } from "@minecraft/server";
import { base64Decode, buildStructure } from "./structure-builder.js";

// Storage for chunked transfers
const chunkSessions = {};

/**
 * Handle incoming chunk data
 * @param {string} message - Format: sessionId:chunkIndex:totalChunks:data
 * @param {Player} player - The player who sent the chunk
 */
function handleChunk(message, player) {
    const colonIdx1 = message.indexOf(":");
    const colonIdx2 = message.indexOf(":", colonIdx1 + 1);
    const colonIdx3 = message.indexOf(":", colonIdx2 + 1);

    const sessionId = message.substring(0, colonIdx1);
    const chunkIndex = parseInt(message.substring(colonIdx1 + 1, colonIdx2));
    const totalChunks = parseInt(message.substring(colonIdx2 + 1, colonIdx3));
    const data = message.substring(colonIdx3 + 1);

    world.sendMessage(`§8[chunk-recv] chunk ${chunkIndex + 1}/${totalChunks} session=${sessionId} (${data.length} chars)`);

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
        world.sendMessage(`§8[chunk-recv] all chunks received, reassembling...`);
        const base64 = session.chunks.join("");
        delete chunkSessions[sessionId];

        const json = base64Decode(base64)
            .map(b => String.fromCharCode(b))
            .join("");
        world.sendMessage(`§8[chunk-recv] decoded JSON (${json.length} chars), parsing...`);
        const structure = JSON.parse(json);

        world.sendMessage(`§8[chunk-recv] structure type=${structure.type}, calling buildStructure`);
        buildStructure(player, structure);
    }
}

/**
 * Initialize the chunk receiver chat message handler
 */
export function initChunkReceiver() {
    world.beforeEvents.chatSend.subscribe((event) => {
        world.sendMessage(`§8[chunk-recv] chatSend: "${event.message.substring(0, 40)}..." from ${event.sender.name}`);
        if (event.message.startsWith("!chunk ")) {
            event.cancel = true;
            const data = event.message.substring(7);
            world.sendMessage(`§8[chunk-recv] processing chunk (${data.length} chars)`);
            system.run(() => {
                try {
                    handleChunk(data, event.sender);
                } catch (e) {
                    world.sendMessage(`§cChunk transfer failed: ${e.message}`);
                }
            });
        }
    });
    world.sendMessage(`§8[chunk-recv] handler registered`);
}
