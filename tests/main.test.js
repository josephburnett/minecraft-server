import { describe, it, expect, beforeAll } from 'vitest';
import { _subscribers, Block } from '../tests/mocks/minecraft-server.js';

// Importing main.js triggers all subscribe() calls
let breakHandler;

beforeAll(async () => {
    // Clear any prior subscribers
    _subscribers.beforePlayerBreakBlock.length = 0;
    await import('../bedrock/behavior_packs/burnodd_scripts/scripts/main.js');
    breakHandler = _subscribers.beforePlayerBreakBlock[0];
});

describe('block protection (playerBreakBlock)', () => {
    it('registers a beforePlayerBreakBlock handler', () => {
        expect(breakHandler).toBeDefined();
        expect(typeof breakHandler).toBe('function');
    });

    it('cancels breaking obsidian', () => {
        const event = {
            block: new Block({ x: 0, y: 0, z: 0 }, 'minecraft:obsidian'),
            cancel: false
        };
        breakHandler(event);
        expect(event.cancel).toBe(true);
    });

    it('does not cancel breaking stone', () => {
        const event = {
            block: new Block({ x: 0, y: 0, z: 0 }, 'minecraft:stone'),
            cancel: false
        };
        breakHandler(event);
        expect(event.cancel).toBe(false);
    });

    it('does not cancel breaking dirt', () => {
        const event = {
            block: new Block({ x: 0, y: 0, z: 0 }, 'minecraft:dirt'),
            cancel: false
        };
        breakHandler(event);
        expect(event.cancel).toBe(false);
    });

    it('does not cancel breaking stone bricks', () => {
        const event = {
            block: new Block({ x: 0, y: 0, z: 0 }, 'minecraft:stone_bricks'),
            cancel: false
        };
        breakHandler(event);
        expect(event.cancel).toBe(false);
    });
});
