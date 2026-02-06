import { describe, it, expect } from 'vitest';
import { ABILITIES } from '../behavior_pack/scripts/abilities.js';
import { Dimension } from './mocks/minecraft-server.js';

// Helper: set up a dimension with blocks in a radius around a center
function setupDimension(center, radius, typeId) {
    const dimension = new Dimension();
    for (let x = -radius; x <= radius; x++) {
        for (let y = -radius; y <= radius; y++) {
            for (let z = -radius; z <= radius; z++) {
                const loc = { x: center.x + x, y: center.y + y, z: center.z + z };
                dimension.setBlock(loc, typeId);
            }
        }
    }
    return dimension;
}

// Helper: create a mock player with a dimension
function mockPlayer(dimension) {
    return {
        dimension,
        sendMessage: () => {},
        runCommand: () => {},
        location: { x: 0, y: 64, z: 0 }
    };
}

// Helper: create a mock blockHit
function mockBlockHit(location, dimension) {
    return {
        block: dimension.getBlock(location),
        faceLocation: location
    };
}

describe('Small Dig (blaze rod)', () => {
    const ability = ABILITIES['minecraft:blaze_rod'];

    it('destroys blocks within radius 2', () => {
        const center = { x: 10, y: 64, z: 10 };
        const dimension = setupDimension(center, 3, 'minecraft:stone');
        const player = mockPlayer(dimension);
        const blockHit = mockBlockHit(center, dimension);

        ability.action(player, blockHit);

        // Center should be air
        expect(dimension.getBlock(center).typeId).toBe('minecraft:air');
        // Block at distance 1 should be air
        expect(dimension.getBlock({ x: 11, y: 64, z: 10 }).typeId).toBe('minecraft:air');
        // Block at distance 3 should be untouched
        expect(dimension.getBlock({ x: 13, y: 64, z: 10 }).typeId).toBe('minecraft:stone');
    });

    it('does not destroy bedrock', () => {
        const center = { x: 10, y: 64, z: 10 };
        const dimension = setupDimension(center, 3, 'minecraft:bedrock');
        const player = mockPlayer(dimension);
        const blockHit = mockBlockHit(center, dimension);

        ability.action(player, blockHit);

        expect(dimension.getBlock(center).typeId).toBe('minecraft:bedrock');
    });

    it('does not destroy obsidian', () => {
        const center = { x: 10, y: 64, z: 10 };
        const dimension = setupDimension(center, 3, 'minecraft:obsidian');
        const player = mockPlayer(dimension);
        const blockHit = mockBlockHit(center, dimension);

        ability.action(player, blockHit);

        expect(dimension.getBlock(center).typeId).toBe('minecraft:obsidian');
        expect(dimension.getBlock({ x: 11, y: 64, z: 10 }).typeId).toBe('minecraft:obsidian');
    });

    it('destroys stone but leaves obsidian intact in mixed terrain', () => {
        const center = { x: 10, y: 64, z: 10 };
        const dimension = setupDimension(center, 3, 'minecraft:stone');
        // Place obsidian at a specific spot within radius
        dimension.setBlock({ x: 11, y: 64, z: 10 }, 'minecraft:obsidian');
        const player = mockPlayer(dimension);
        const blockHit = mockBlockHit(center, dimension);

        ability.action(player, blockHit);

        expect(dimension.getBlock(center).typeId).toBe('minecraft:air');
        expect(dimension.getBlock({ x: 11, y: 64, z: 10 }).typeId).toBe('minecraft:obsidian');
    });
});

describe('Big Dig (fire charge)', () => {
    const ability = ABILITIES['minecraft:fire_charge'];

    it('destroys blocks within radius 8', () => {
        const center = { x: 10, y: 64, z: 10 };
        const dimension = setupDimension(center, 9, 'minecraft:stone');
        const player = mockPlayer(dimension);
        const blockHit = mockBlockHit(center, dimension);

        ability.action(player, blockHit);

        // Center should be air
        expect(dimension.getBlock(center).typeId).toBe('minecraft:air');
        // Block at distance 7 should be air
        expect(dimension.getBlock({ x: 17, y: 64, z: 10 }).typeId).toBe('minecraft:air');
        // Block at distance 9 should be untouched
        expect(dimension.getBlock({ x: 19, y: 64, z: 10 }).typeId).toBe('minecraft:stone');
    });

    it('does not destroy bedrock', () => {
        const center = { x: 10, y: 64, z: 10 };
        const dimension = setupDimension(center, 9, 'minecraft:bedrock');
        const player = mockPlayer(dimension);
        const blockHit = mockBlockHit(center, dimension);

        ability.action(player, blockHit);

        expect(dimension.getBlock(center).typeId).toBe('minecraft:bedrock');
    });

    it('does not destroy obsidian', () => {
        const center = { x: 10, y: 64, z: 10 };
        const dimension = setupDimension(center, 9, 'minecraft:obsidian');
        const player = mockPlayer(dimension);
        const blockHit = mockBlockHit(center, dimension);

        ability.action(player, blockHit);

        expect(dimension.getBlock(center).typeId).toBe('minecraft:obsidian');
    });
});
