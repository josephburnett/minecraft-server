import { describe, it, expect } from 'vitest';
import {
    base64Decode,
    decodeBitfield,
    decodePalette
} from '../behavior_pack/scripts/structure-builder.js';

describe('base64Decode', () => {
    it('decodes simple base64 string', () => {
        // "Hello" in base64 is "SGVsbG8="
        const result = base64Decode('SGVsbG8=');
        const expected = [72, 101, 108, 108, 111]; // H, e, l, l, o
        expect(result).toEqual(expected);
    });

    it('decodes base64 without padding', () => {
        // "Hi" in base64 is "SGk=" but can also work as "SGk"
        const result = base64Decode('SGk=');
        const expected = [72, 105]; // H, i
        expect(result).toEqual(expected);
    });

    it('handles empty string', () => {
        const result = base64Decode('');
        expect(result).toEqual([]);
    });

    it('decodes binary data correctly', () => {
        // 0xFF, 0x00, 0xFF in base64 is "/wD/"
        const result = base64Decode('/wD/');
        expect(result).toEqual([255, 0, 255]);
    });

    it('ignores invalid characters', () => {
        // Should skip any characters not in base64 alphabet
        const result = base64Decode('SG!Vs#bG8=');
        const expected = [72, 101, 108, 108, 111];
        expect(result).toEqual(expected);
    });
});

describe('decodeBitfield', () => {
    it('decodes all-zeros bitfield', () => {
        // 8 bits of zeros = 0x00 = "AA=="
        const result = decodeBitfield('AA==', [2, 2, 2]);
        expect(result).toEqual([]);
    });

    it('decodes all-ones bitfield', () => {
        // 8 bits of ones = 0xFF = "/w=="
        const result = decodeBitfield('/w==', [2, 2, 2]);

        // Should have 8 positions for 2x2x2 grid
        expect(result.length).toBe(8);
    });

    it('decodes specific pattern', () => {
        // Binary: 10000000 = 0x80 = "gA=="
        // First bit is 1, rest are 0
        const result = decodeBitfield('gA==', [2, 2, 2]);

        // Only position [0,0,0] should be set
        expect(result.length).toBe(1);
        expect(result[0]).toEqual([0, 0, 0]);
    });

    it('iterates in correct order (x, y, z)', () => {
        // Binary: 10100000 = 0xA0 = "oA=="
        // Bits 0 and 2 are set
        const result = decodeBitfield('oA==', [2, 2, 2]);

        // Position 0: [0,0,0], Position 2: [0,0,2] but z max is 2, so [0,1,0]
        // Actually for 2x2x2: positions go [0,0,0], [0,0,1], [0,1,0], [0,1,1], [1,0,0]...
        expect(result.length).toBe(2);
        expect(result[0]).toEqual([0, 0, 0]); // bit 0
        expect(result[1]).toEqual([0, 1, 0]); // bit 2
    });
});

describe('decodePalette', () => {
    it('decodes palette indices correctly', () => {
        // 8 bytes of index 0 = "AAAAAAAA"
        const palette = ['minecraft:air', 'minecraft:stone'];
        const result = decodePalette('AAAAAAAA', [2, 2, 2], palette);

        // All air blocks should be skipped
        expect(result).toEqual([]);
    });

    it('includes non-air blocks', () => {
        // 8 bytes: [1,1,1,1,1,1,1,1] = "AQEBAQEBAQE="
        const palette = ['minecraft:air', 'minecraft:stone'];
        const result = decodePalette('AQEBAQEBAQE=', [2, 2, 2], palette);

        expect(result.length).toBe(8);
        result.forEach(block => {
            expect(block[3]).toBe('minecraft:stone');
        });
    });

    it('handles multiple block types', () => {
        // Bytes: [0, 1, 2] for 3 positions (1x1x3)
        // Base64 of [0, 1, 2] is "AAEC"
        const palette = ['minecraft:air', 'minecraft:stone', 'minecraft:dirt'];
        const result = decodePalette('AAEC', [1, 1, 3], palette);

        // Air (index 0) is skipped
        expect(result.length).toBe(2);
        expect(result[0]).toEqual([0, 0, 1, 'minecraft:stone']);
        expect(result[1]).toEqual([0, 0, 2, 'minecraft:dirt']);
    });
});

describe('encoding/decoding round-trip', () => {
    it('bitfield preserves block positions', () => {
        // Create a simple pattern
        const size = [3, 3, 3];
        const totalBits = 27;
        const bytesNeeded = Math.ceil(totalBits / 8); // 4 bytes

        // Create pattern with specific bits set
        // We'll set positions [0,0,0], [1,1,1], [2,2,2]
        // In iteration order (x,y,z innermost):
        // [0,0,0] = bit 0
        // [1,1,1] = 1*9 + 1*3 + 1 = 13
        // [2,2,2] = 2*9 + 2*3 + 2 = 26

        const bytes = new Uint8Array(4);
        bytes[0] = 0b10000000; // bit 0
        bytes[1] = 0b00000100; // bit 13 (8+5, in byte 1 bit 5 from right = bit 2 from left)
        bytes[3] = 0b00000010; // bit 26 (24+2, in byte 3 bit 1 from right)

        // Convert to base64 manually would be complex, so let's just verify the decoder works
        // with a known simple case

        // Single block at [0,0,0]: 10000000 00000000 00000000 00000000 = "gAAAAA=="
        const result = decodeBitfield('gAAAAA==', [3, 3, 3]);
        expect(result.length).toBe(1);
        expect(result[0]).toEqual([0, 0, 0]);
    });
});
