import fs from '@ohos.file.fs';
import { Context } from '@kit.AbilityKit';
import { getProfileDir, getProfilePath } from './appPath';
import { YamlUtils } from './utils/YamlUtils';
import { stringify } from 'yaml';

export class TailscaleProfileSettings {
  name: string = 'Embedded Tailscale'
  hostname: string = 'clashbox-fold'
  stateDir: string = 'tailscale-fold'
  controlUrl: string = 'https://controlplane.tailscale.com'
  acceptRoutes: boolean = true
  authKey: string = ''
}

export interface TailscaleNodeStatus {
  state: string
  authURL: string
  ips: string[]
  peers: number
  health: string[]
}

export class TailscaleProfile {
  static async read(context: Context, id: string): Promise<TailscaleProfileSettings> {
    const settingsPath = await getProfileDir(context, id) + '/tailscale-settings.json';
    if (await fs.access(settingsPath)) {
      return JSON.parse(await fs.readText(settingsPath)) as TailscaleProfileSettings;
    }
    const result = new TailscaleProfileSettings();
    const path = await getProfilePath(context, id);
    if (await fs.access(path)) {
      const doc = YamlUtils.parseYamlSafe(await fs.readText(path));
      const proxies = doc['proxies'] as Record<string, Object | undefined>[] | undefined;
      const existing = proxies?.find((p) => p['type'] === 'tailscale');
      if (existing) {
        result.name = existing['name'] as string;
        result.hostname = existing['hostname'] as string || result.hostname;
        result.stateDir = existing['state-dir'] as string || result.stateDir;
        result.controlUrl = existing['control-url'] as string || result.controlUrl;
        result.acceptRoutes = existing['accept-routes'] !== false;
        // Existing enrollment credentials stay in the profile; never copy them to a text field.
      }
    }
    return result;
  }

  static transform(yaml: string, settings: TailscaleProfileSettings): string {
    if (!settings.name.trim() || !settings.hostname.trim() || !settings.stateDir.trim()) {
      throw new Error('Tailscale name, hostname and state directory are required');
    }
    if (!settings.controlUrl.startsWith('https://')) {
      throw new Error('The control server must use HTTPS');
    }
    const doc = YamlUtils.parseYamlSafe(yaml);
    const proxies = doc['proxies'] as Record<string, Object | undefined>[] | undefined;
    if (!proxies || !Array.isArray(proxies)) throw new Error('This profile has no proxy list');
    let node = proxies.find((p) => p['name'] === settings.name);
    if (node && node['type'] !== 'tailscale') throw new Error('This name belongs to another proxy');
    if (!node) {
      node = { 'name': settings.name, 'type': 'tailscale' };
      proxies.push(node);
    }
    node['hostname'] = settings.hostname;
    node['state-dir'] = settings.stateDir;
    node['control-url'] = settings.controlUrl;
    node['accept-routes'] = settings.acceptRoutes;
    node['ephemeral'] = false;
    node['udp'] = true;
    // Tailscale reaches its control server and peers directly; it cannot dial through itself.
    node['dialer-proxy'] = undefined;
    if (settings.authKey.trim()) node['auth-key'] = settings.authKey.trim();
    for (const proxy of proxies) {
      if (proxy['type'] === 'tailscale') continue;
      const server = proxy['server'] as string | undefined;
      if (!server) continue;
      const octets = server.split('.').map((value) => Number(value));
      if (octets.length === 4 && octets[0] === 100 && octets[1] >= 64 && octets[1] <= 127 &&
        (!proxy['dialer-proxy'] || proxy['dialer-proxy'] === settings.name)) {
        proxy['dialer-proxy'] = settings.name;
      }
    }
    const rules = doc['rules'] as string[] | undefined || [];
    const tailnetRule = `IP-CIDR,100.64.0.0/10,${settings.name},no-resolve`;
    doc['rules'] = [tailnetRule, ...rules.filter((rule) => rule !== tailnetRule)];
    return stringify(doc);
  }

  private static write(path: string, content: string): void {
    const file = fs.openSync(path + '.tmp', fs.OpenMode.CREATE | fs.OpenMode.READ_WRITE | fs.OpenMode.TRUNC);
    try { fs.writeSync(file.fd, content); fs.fsyncSync(file.fd); } finally { fs.closeSync(file); }
    fs.renameSync(path + '.tmp', path);
  }

  static async save(context: Context, id: string, settings: TailscaleProfileSettings): Promise<void> {
    const path = await getProfilePath(context, id);
    const original = await fs.readText(path);
    const updated = TailscaleProfile.transform(original, settings);
    const backup = path + '.before-tailscale';
    if (!await fs.access(backup)) TailscaleProfile.write(backup, original);
    TailscaleProfile.write(await getProfileDir(context, id) + '/tailscale-settings.json', JSON.stringify(settings));
    TailscaleProfile.write(path, updated);
  }

  static async applySaved(context: Context, id: string): Promise<void> {
    const settingsPath = await getProfileDir(context, id) + '/tailscale-settings.json';
    if (!await fs.access(settingsPath)) return;
    const settings = JSON.parse(await fs.readText(settingsPath)) as TailscaleProfileSettings;
    const path = await getProfilePath(context, id);
    const original = await fs.readText(path);
    const updated = TailscaleProfile.transform(original, settings);
    if (updated !== original) TailscaleProfile.write(path, updated);
  }
}
