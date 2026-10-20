const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const { stripTypeScriptTypes } = require('node:module');
// Exercise the production transformation with JSON serialization standing in
// for YAML. This checks routing and credential semantics independently of IO.
let source = fs.readFileSync('proxy_core/src/main/ets/TailscaleProfile.ets', 'utf8')
  .replace(/^import .*;\n/gm, '').replace(/^export /gm, '');
source = stripTypeScriptTypes(source);
const sandbox = { YamlUtils: { parseYamlSafe: JSON.parse }, stringify: JSON.stringify };
vm.runInNewContext(source + '\nglobalThis.transform = TailscaleProfile.transform; globalThis.Settings = TailscaleProfileSettings; globalThis.Profile = TailscaleProfile;', sandbox);
const settings = () => new sandbox.Settings();
const profile = () => ({ proxies: [
  { name: 'US', type: 'hysteria2', server: '100.68.117.89', port: 3446, password: 'private-password', sni: 'existing.example' },
  { name: 'China', type: 'socks5', server: 'china.example', port: 1080 },
], dns: { nameserver: ['223.5.5.5'] }, rules: ['DOMAIN-SUFFIX,baidu.com,China', 'MATCH,US'] });
const apply = (doc, opts = settings()) => JSON.parse(sandbox.transform(JSON.stringify(doc), opts));
test('chains tailnet Hysteria without changing protocol, credentials, China DNS or policy', () => {
  const before = profile(), after = apply(before);
  assert.deepEqual(after.proxies[0], { ...before.proxies[0], 'dialer-proxy': 'Embedded Tailscale' });
  assert.deepEqual(after.proxies[1], before.proxies[1]);
  assert.deepEqual(after.dns, before.dns);
  assert.deepEqual(after.rules.slice(1), before.rules);
  assert.equal(after.rules[0], 'IP-CIDR,100.64.0.0/10,Embedded Tailscale,no-resolve');
});
test('preserves existing enrolled identity, key, unknown options and explicit dialer policy', () => {
  const before = profile(); before.proxies[0]['dialer-proxy'] = 'Explicit custom route';
  before.proxies.push({ name: 'Embedded Tailscale', type: 'tailscale', 'auth-key': 'keep-key', 'state-dir': 'existing-state', 'custom-option': true, 'dialer-proxy': 'US' });
  const opts = settings(); opts.stateDir = 'existing-state';
  const after = apply(before, opts), ts = after.proxies[2];
  assert.equal(ts['auth-key'], 'keep-key'); assert.equal(ts['state-dir'], 'existing-state');
  assert.equal(ts['custom-option'], true); assert.equal(ts['dialer-proxy'], undefined);
  assert.equal(after.proxies[0]['dialer-proxy'], 'Explicit custom route');
  assert.deepEqual(apply(after, opts), after);
});
test('new enrollment key replaces only Tailscale auth key', () => {
  const opts = settings(); opts.authKey = ' new-key ';
  const after = apply(profile(), opts);
  assert.equal(after.proxies[2]['auth-key'], 'new-key');
  assert.equal(after.proxies[0].password, 'private-password');
});
test('refuses a conflicting proxy name and non-HTTPS controller', () => {
  const opts = settings(); opts.name = 'US'; assert.throws(() => apply(profile(), opts), /belongs to another proxy/);
  opts.name = 'TS'; opts.controlUrl = 'http://unsafe'; assert.throws(() => apply(profile(), opts), /HTTPS/);
});

test('save/read/reapply survives subscription replacement and preserves first backup', async () => {
  const path = require('node:path'), os = require('node:os');
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'clashbox-profile-test-'));
  const configPath = path.join(dir, 'config.yaml');
  const shim = {
    access: async (p) => fs.existsSync(p), readText: async (p) => fs.readFileSync(p, 'utf8'),
    openSync: (p) => ({ fd: fs.openSync(p, 'w') }), writeSync: fs.writeSync,
    fsyncSync: fs.fsyncSync, closeSync: (file) => fs.closeSync(file.fd), renameSync: fs.renameSync,
    OpenMode: { CREATE: 1, READ_WRITE: 2, TRUNC: 4 },
  };
  sandbox.fs = shim; sandbox.getProfileDir = async () => dir; sandbox.getProfilePath = async () => configPath;
  try {
    const original = JSON.stringify(profile()); fs.writeFileSync(configPath, original);
    const opts = settings(); opts.authKey = 'stored-enrollment'; opts.stateDir = 'keep-enrolled-state';
    await sandbox.Profile.save({}, 'active-profile', opts);
    assert.equal((await sandbox.Profile.read({}, 'active-profile')).stateDir, 'keep-enrolled-state');
    const replacement = profile(); replacement.proxies[0].password = 'new-subscription-password';
    fs.writeFileSync(configPath, JSON.stringify(replacement));
    await sandbox.Profile.applySaved({}, 'active-profile');
    const after = JSON.parse(fs.readFileSync(configPath, 'utf8'));
    assert.equal(after.proxies[0].password, 'new-subscription-password');
    assert.equal(after.proxies[0]['dialer-proxy'], opts.name);
    assert.equal(after.proxies[2]['auth-key'], 'stored-enrollment');
    assert.equal(after.proxies[2]['state-dir'], 'keep-enrolled-state');
    assert.equal(fs.readFileSync(configPath + '.before-tailscale', 'utf8'), original);
  } finally { fs.rmSync(dir, { recursive: true, force: true }); }
});
