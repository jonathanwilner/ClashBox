const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const { stripTypeScriptTypes } = require('node:module');
function method(file, signature) {
  const source = fs.readFileSync(file, 'utf8'), start = source.indexOf(signature);
  assert.ok(start >= 0, signature);
  let end = source.indexOf('{', start), level = 1;
  while (level) { end++; if (source[end] === '{') level++; else if (source[end] === '}') level--; }
  return source.slice(start, end + 1).replace(/\b(private|override) /g, '');
}
function probe(body, globals = {}, parent = '') {
  const box = { ...globals };
  vm.runInNewContext(stripTypeScriptTypes(`${parent}\nclass Probe ${parent ? 'extends Parent' : ''} { ${body} }\nglobalThis.Probe = Probe;`), box);
  return new box.Probe();
}
const page = 'entry/src/main/ets/components/Settings/TailscaleSettings.ets';
test('settings navigation exposes the compiled Tailscale page and browser enrollment action', () => {
  const navigation = fs.readFileSync('entry/src/main/ets/components/Settings/Settings.ets', 'utf8');
  assert.match(navigation, /Button\('Tailscale · 内置组网'\)/);
  assert.match(navigation, /pushPathByName\('Tailscale', false\)/);
  assert.match(navigation, /name == 'Tailscale'[\s\S]*?TailscaleSettings\(\)/);
  assert.match(fs.readFileSync(page, 'utf8'), /action: 'ohos.want.action.viewData', uri: this.authURL/);
});
test('production status action displays native state, IPs and enrollment link, including RPC string errors', async () => {
  let reply = JSON.stringify({ state: 'NeedsLogin', ips: [], peers: 0, authURL: 'https://login.tailscale.com/example' });
  const model = { socketProxy: { tailscaleStatus: async () => reply } };
  const p = probe(method(page, '  private async queryStatus('), { ClashViewModel: model });
  p.nodeName = 'TS'; await p.queryStatus(true);
  assert.match(p.status, /NeedsLogin/); assert.equal(p.authURL, 'https://login.tailscale.com/example');
  reply = JSON.stringify({ state: 'Running', ips: ['100.64.1.2'], peers: 3, health: [] });
  await p.queryStatus(false); assert.match(p.status, /Running\n100.64.1.2\n3 peers/); assert.equal(p.authURL, '');
  model.socketProxy.tailscaleStatus = async () => { throw 'native enrollment unavailable'; };
  await p.queryStatus(false); assert.match(p.status, /native enrollment unavailable/); assert.equal(p.busy, false);
});
test('VPN start waits for protect readiness before recreating profile adapters', async () => {
  const events = []; let ready;
  const started = new Promise((resolve) => ready = resolve);
  const p = probe(method('entry/src/main/ets/entryability/ClashViewModel.ets', '  async StartVpn(){'), {
    EventHub: { sendEvent() {}, on() {} }, EventKey: {}, hilog: { info() {} },
    cardManager: { pushCartVpnServiceTime() {}, pushCartProxyMode() {}, pushCartVpnServiceTime() {} },
    PROXY_STARTED_DURATION_INIT_VALUE: 0, number2Time: () => '',
  });
  p.loadVpnOptions = async () => events.push('options');
  p.socketProxy = { startClash: async () => { events.push('start'); await started; return true; } };
  p.loadConfig = async () => events.push('reload'); p.getRuntime = async () => 1;
  p.proxyStartedTimer = { start() {}, reset() {} };
  const running = p.StartVpn(); await new Promise(setImmediate);
  assert.deepEqual(events, ['options', 'start']); ready(); await running;
  assert.deepEqual(events, ['options', 'start', 'reload']);
});
test('VPN service reports success only after native protect acknowledgement', async () => {
  let ready; const acknowledged = new Promise((resolve) => ready = resolve);
  const p = probe(method('proxy_core/src/main/ets/rpc/FlClashVpnService.ets', '  override async startVpn()'), {},
    'class Parent { async getTunFd() { return 42; } }');
  p.ParseConfig = () => ({}); p.startClash = async () => acknowledged;
  let finished = false; const running = p.startVpn().then((result) => { finished = true; return result; });
  await new Promise(setImmediate); assert.equal(finished, false);
  ready(); assert.equal(await running, true);
});
test('real profile-load rejection propagates through loadConfig and stops the newly acquired VPN', async () => {
  const events = [], file = 'entry/src/main/ets/entryability/ClashViewModel.ets';
  const body = ['  async StartVpn(){', '  async loadConfig(', '  async StopVpn(){'].map((s) => method(file, s)).join('\n');
  const p = probe(body, {
    EventHub: { sendEvent(key) { events.push(key); }, on() {} },
    EventKey: { StartedClash: 'started' }, hilog: { info() {} }, console: { error() {}, log() {} },
    AppStorage: { get() { return { currentProfileId: 'active' }; } }, AppConfig: class {}, ClashConfig: class {},
    cardManager: { pushCartProxyMode() {} }, TailscaleProfile: { async applySaved() {} },
  });
  p.context = {}; p.profileRepo = { async query() { return { id: 'active', getSelectedMap: () => ({}) }; } };
  p.loadVpnOptions = async () => {};
  p.loadProfileAndConfig = async () => { throw new Error('native profile load failed'); };
  p.socketProxy = { async startClash() { return true; }, async stopClash() { events.push('stopped'); } };
  p.proxyStartedTimer = { reset() {}, start() { assert.fail('failed profile must not start timer'); } };
  await assert.rejects(p.StartVpn(), /native profile load failed/);
  assert.equal(p.vpnStarted, false); assert.ok(events.includes('stopped')); assert.ok(!events.includes('started'));
});
