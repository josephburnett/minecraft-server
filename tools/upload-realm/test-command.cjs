#!/usr/bin/env node
/**
 * Track A: Test if bedrock-protocol can send CommandRequest packets to a Realm
 * without triggering PacketViolationWarning.
 *
 * Usage: node tools/upload-realm/test-command.js
 *
 * Requires: REALM_INVITE env var or .realm-invite file in project root
 */

const { createClient } = require('bedrock-protocol');
const fs = require('fs');
const path = require('path');

// Read realm invite code
function getRealmInvite() {
  if (process.env.REALM_INVITE) return process.env.REALM_INVITE;
  const inviteFile = path.join(__dirname, '..', '..', '.realm-invite');
  if (fs.existsSync(inviteFile)) {
    return fs.readFileSync(inviteFile, 'utf-8').trim();
  }
  throw new Error('No realm invite found. Set REALM_INVITE env var or create .realm-invite file.');
}

const realmInvite = getRealmInvite();
console.log(`Realm invite code: ${realmInvite}`);

console.log('Connecting to Realm via bedrock-protocol...');
console.log('(Microsoft auth will open in browser if needed)');

const client = createClient({
  realms: {
    realmInvite: realmInvite,
  },
  raknetBackend: 'jsp-raknet',
  // profilesFolder: path.join(__dirname, '..', '..'),
  // Use default auth flow (will open browser for Microsoft login)
});

let spawned = false;

client.on('connect', () => {
  console.log('[connect] Connected to server');
});

client.on('join', () => {
  console.log('[join] Joined game');
});

client.on('spawn', () => {
  spawned = true;
  console.log('[spawn] Player spawned!');
  console.log('Waiting 3 seconds before sending /help...');

  setTimeout(() => {
    console.log('[3s] Sending /help command...');
    client.queue('command_request', {
      command: '/help',
      origin: {
        type: 'player',
        uuid: '',
        request_id: '',
      },
      internal: false,
      version: 72,
    });
    console.log('[3s] /help sent');
  }, 3000);

  setTimeout(() => {
    console.log('[6s] Sending /scriptevent test...');
    client.queue('command_request', {
      command: '/scriptevent burnodd:chunk test:0:1:dGVzdA==',
      origin: {
        type: 'player',
        uuid: '',
        request_id: '',
      },
      internal: false,
      version: 72,
    });
    console.log('[6s] /scriptevent sent');
  }, 6000);

  // Disconnect after 15 seconds
  setTimeout(() => {
    console.log('[15s] Test complete, disconnecting...');
    client.close();
    process.exit(0);
  }, 15000);
});

client.on('packet', (packet) => {
  // Log interesting packets
  const name = packet.data?.name || packet.name;
  if (!name) return;
});

client.on('command_output', (packet) => {
  console.log('[CommandOutput]', JSON.stringify(packet, null, 2));
});

client.on('packet_violation_warning', (packet) => {
  console.log('[ViolationWarning]', JSON.stringify(packet, null, 2));
});

client.on('text', (packet) => {
  console.log(`[Text] type=${packet.type} message=${packet.message}`);
});

client.on('script_message', (packet) => {
  console.log(`[ScriptMessage] id=${packet.message_id} data=${packet.message}`);
});

client.on('disconnect', (packet) => {
  console.log('[Disconnect]', JSON.stringify(packet, null, 2));
});

client.on('error', (err) => {
  console.error('[Error]', err.message || err);
});

client.on('close', () => {
  console.log('[Close] Connection closed');
  process.exit(0);
});

// Timeout if we can't connect
setTimeout(() => {
  if (!spawned) {
    console.error('Timeout: Failed to spawn within 60 seconds');
    client.close();
    process.exit(1);
  }
}, 60000);
