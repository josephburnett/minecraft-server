/**
 * Mock for @minecraft/server module
 * Provides minimal implementations for testing
 */

// Subscriber store — tests can inspect registered callbacks
export const _subscribers = {
  beforePlayerBreakBlock: [],
  beforeChatSend: [],
  afterItemUse: [],
  afterWorldLoad: []
};

// Mock world object
export const world = {
  getAllPlayers: () => [],
  sendMessage: (msg) => console.log('[world]', msg),
  beforeEvents: {
    playerBreakBlock: {
      subscribe: (callback) => { _subscribers.beforePlayerBreakBlock.push(callback); }
    },
    chatSend: {
      subscribe: (callback) => { _subscribers.beforeChatSend.push(callback); }
    }
  },
  afterEvents: {
    itemUse: {
      subscribe: (callback) => { _subscribers.afterItemUse.push(callback); }
    },
    worldLoad: {
      subscribe: (callback) => { _subscribers.afterWorldLoad.push(callback); }
    }
  }
};

// Mock system object
export const system = {
  runInterval: (callback, interval) => {
    const id = setInterval(callback, interval);
    return id;
  },
  clearRun: (id) => {
    clearInterval(id);
  },
  run: (callback) => {
    callback();
  }
};

// Mock player class
export class Player {
  constructor(name = 'TestPlayer', x = 0, y = 64, z = 0) {
    this.name = name;
    this.location = { x, y, z };
    this.dimension = new Dimension();
    this._messages = [];
    this._commands = [];
  }

  sendMessage(msg) {
    this._messages.push(msg);
  }

  runCommand(cmd) {
    this._commands.push(cmd);
  }

  getBlockFromViewDirection(options) {
    return null;
  }
}

// Mock dimension class
export class Dimension {
  constructor() {
    this._blocks = new Map();
  }

  getBlock(location) {
    const key = `${location.x},${location.y},${location.z}`;
    if (!this._blocks.has(key)) {
      this._blocks.set(key, new Block(location, 'minecraft:air'));
    }
    return this._blocks.get(key);
  }

  setBlock(location, typeId) {
    const key = `${location.x},${location.y},${location.z}`;
    this._blocks.set(key, new Block(location, typeId));
  }
}

// Mock block class
export class Block {
  constructor(location, typeId = 'minecraft:air') {
    this.location = location;
    this.typeId = typeId;
  }

  setType(typeId) {
    this.typeId = typeId;
  }
}
