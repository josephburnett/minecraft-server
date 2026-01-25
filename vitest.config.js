import { defineConfig } from 'vitest/config';
import { fileURLToPath } from 'url';
import { dirname, resolve } from 'path';

const __dirname = dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  resolve: {
    alias: {
      '@minecraft/server': resolve(__dirname, 'tests/mocks/minecraft-server.js')
    }
  },
  test: {
    globals: true,
    environment: 'node',
    include: ['tests/**/*.test.js']
  }
});
