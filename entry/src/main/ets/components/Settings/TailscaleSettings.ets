import { common } from '@kit.AbilityKit';
import { TailscaleProfile, TailscaleProfileSettings, TailscaleNodeStatus, ClashConfig } from 'proxy_core';
import TopBuilder from './TopBuilder';
import ClashViewModel from '../../entryability/ClashViewModel';
import { AppConfig } from '../../entryability/AppState';

@Component
export default struct TailscaleSettings {
  @Consume('NavSettingsPathStack') SettingsPageInfos: NavPathStack
  @StorageLink('appConfig') appConfig: AppConfig = new AppConfig()
  @State private nodeName: string = 'Embedded Tailscale'
  @State private hostname: string = 'clashbox-fold'
  @State private stateDir: string = 'tailscale-fold'
  @State private controlUrl: string = 'https://controlplane.tailscale.com'
  @State private authKey: string = ''
  @State private acceptRoutes: boolean = true
  @State private status: string = 'Apply the profile, then check connection status.'
  @State private authURL: string = ''
  @State private busy: boolean = false
  private settings: TailscaleProfileSettings = new TailscaleProfileSettings()
  private profileId: string = ''
  private context = this.getUIContext().getHostContext()! as common.UIAbilityContext

  async aboutToAppear(): Promise<void> {
    this.profileId = this.appConfig.currentProfileId || '';
    if (!this.profileId) {
      this.status = 'Select a profile on the Profiles page first.';
      return;
    }
    try {
      this.settings = await TailscaleProfile.read(this.context, this.profileId);
      this.nodeName = this.settings.name;
      this.hostname = this.settings.hostname;
      this.stateDir = this.settings.stateDir;
      this.controlUrl = this.settings.controlUrl;
      this.acceptRoutes = this.settings.acceptRoutes;
    } catch (e) {
      this.status = `Cannot read the active profile: ${(e as Error).message || String(e)}`;
    }
  }

  private async apply(): Promise<void> {
    this.busy = true;
    try {
      if (!this.profileId || this.profileId !== this.appConfig.currentProfileId) {
        throw new Error('Select a profile, then reopen Tailscale settings');
      }
      // Retain the existing node identity and key unless the user supplies a replacement.
      this.settings.name = this.nodeName.trim();
      this.settings.hostname = this.hostname.trim();
      this.settings.stateDir = this.stateDir.trim();
      this.settings.controlUrl = this.controlUrl.trim();
      this.settings.acceptRoutes = this.acceptRoutes;
      if (this.authKey.trim()) this.settings.authKey = this.authKey.trim();
      await TailscaleProfile.save(this.context, this.profileId, this.settings);
      this.authKey = '';
      const profile = await ClashViewModel.getProfile(this.profileId);
      await ClashViewModel.loadProfileAndConfig(this.profileId,
        AppStorage.get<ClashConfig>('clashConfig') || new ClashConfig(), { 'is-patch': false, 'is-compatible': true,
          'override-dns': AppStorage.get<ClashConfig>('clashConfig')?.overrideDns ?? false,
          'selected-map': profile?.getSelectedMap() || {}, 'test-url': '' });
      await this.queryStatus(false);
    } catch (e) {
      this.status = `Tailscale setup failed: ${(e as Error).message || String(e)}`;
    } finally { this.busy = false; }
  }

  private async queryStatus(login: boolean): Promise<void> {
    this.busy = true;
    try {
      const result = await ClashViewModel.socketProxy.tailscaleStatus(this.nodeName, login);
      const state = JSON.parse(result) as TailscaleNodeStatus;
      this.authURL = state.authURL || '';
      this.status = `${state.state}\n${state.ips?.join(', ') || 'No Tailscale address yet'}\n` +
        `${state.peers || 0} peers\n${state.health?.join('\n') || ''}`;
      if (login && !this.authURL && state.state !== 'Running') {
        this.status += '\nPreparing sign-in. Tap Check status to retrieve the login link.';
      }
    } catch (e) {
      this.status = `Connection unavailable: ${(e as Error).message || String(e)}`;
    } finally { this.busy = false; }
  }

  build() {
    NavDestination() {
      TopBuilder({ title: 'Tailscale · 内置组网', pathStack: this.SettingsPageInfos })
      Scroll() {
        Column({ space: 12 }) {
          Text(`Active profile: ${this.appConfig.currentProfileName || 'None'}`).fontWeight(FontWeight.Bold)
          Text('Connect this profile to your tailnet. Existing China rules and proxy passwords are preserved.')
          Text('Node name · 节点名称').width('100%')
          Text(this.nodeName).width('100%')
          Text('Device hostname · 设备名称').width('100%')
          TextInput({ text: this.hostname }).onChange((value) => this.hostname = value)
          Text('State directory · 身份保存目录').width('100%')
          Text(this.stateDir).width('100%')
          Text('Keep this directory unchanged to retain the enrolled device.').fontSize(13)
          Text('Control server · 控制服务器').width('100%')
          TextInput({ text: this.controlUrl }).onChange((value) => this.controlUrl = value)
          Text('Enrollment key · 注册密钥').width('100%')
          TextInput({ text: this.authKey, placeholder: 'Optional. Leave blank to keep existing enrollment.' })
            .type(InputType.Password).onChange((value) => this.authKey = value)
          Row() {
            Text('Accept subnet routes · 接受子网路由').layoutWeight(1)
            Toggle({ type: ToggleType.Switch, isOn: this.acceptRoutes })
              .onChange((value) => this.acceptRoutes = value)
          }.width('100%')
          Button('Apply to active profile · 应用到当前配置').enabled(!this.busy && !!this.profileId)
            .onClick(() => this.apply())
          Button('Check status · 检查状态').enabled(!this.busy).onClick(() => this.queryStatus(false))
          Button('Sign in · 登录').enabled(!this.busy).onClick(() => this.queryStatus(true))
          Text(this.status).width('100%').copyOption(CopyOptions.InApp)
          if (this.authURL) {
            Button('Open Tailscale sign-in · 打开登录页面').onClick(async () => {
              try {
                await this.context.startAbility({ action: 'ohos.want.action.viewData', uri: this.authURL });
              } catch (e) { this.status = `Cannot open sign-in: ${(e as Error).message || String(e)}`; }
            })
          }
          Text('Tailnet proxy servers use this Tailscale connection. Start ClashBox VPN on the Home page for automatic app routing.')
            .fontSize(13)
        }.padding(20).width('100%')
      }.layoutWeight(1)
    }.hideTitleBar(true)
  }
}
