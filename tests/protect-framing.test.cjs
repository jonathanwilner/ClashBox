const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const { stripTypeScriptTypes } = require('node:module');

function service() {
  let source = fs.readFileSync('proxy_core/src/main/ets/rpc/FlClashVpnService.ets', 'utf8')
    .replace(/^import[\s\S]*?from\s+['"][^'"]+['"];?/gm, '').replace(/^export /gm, '');
  source = stripTypeScriptTypes(source);
  const handlers = {};
  const sent = [];
  const channel = {
    on: (event, fn) => { handlers[event] = fn; }, off: () => {}, close: () => {},
    connect: async () => {}, send: async (value) => { sent.push(value); }
  };
  class Decoder {
    decoder = new TextDecoder();
    decodeToString(value, options) { return this.decoder.decode(value, options); }
  }
  const sandbox = {
    CommonVpnService: class { context = { filesDir: '/private' }; },
    util: { TextDecoder: Decoder }, socket: { constructLocalSocketInstance: () => channel },
    ClashRpcType: { startClash: 13 }, JSON, console, setTimeout, clearTimeout
  };
  vm.runInNewContext(source + '\nglobalThis.Instance = FlClashVpnService;', sandbox);
  const instance = new sandbox.Instance();
  const protectedFrames = [];
  instance.handleProtectMessage = (frame) => protectedFrames.push(JSON.parse(frame));
  const feed = (text) => handlers.message({ message: new TextEncoder().encode(text).buffer });
  return { instance, protectedFrames, feed, sent };
}

test('production protect stream retains split ready delimiter and multiple fragmented fd frames', async () => {
  const { instance, protectedFrames, feed, sent } = service();
  let resolved = false;
  const started = instance.startClash(42).then(() => { resolved = true; });
  await Promise.resolve();
  assert.equal(sent.length, 1);
  assert.equal(JSON.parse(sent[0].data).params[0], 42);
  feed('{"result":"rea');
  feed('dy"}EO');
  await Promise.resolve();
  assert.equal(resolved, false, 'must await complete native ready acknowledgment');
  feed('F{"result":"{\\"id\\":1,\\"value\\":101}"}EOF{"result":"{\\"id\\":2,');
  await started;
  feed('\\"value\\":102}"}EOF');
  assert.deepEqual(protectedFrames.map((f) => JSON.parse(f.result)), [{ id: 1, value: 101 }, { id: 2, value: 102 }]);
});
