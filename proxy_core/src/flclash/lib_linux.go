//go:build ohos && cgo

package main

/*
#include <stdlib.h>
#include <net/if.h>
*/
import "C"
import (
	"core/platform"
	"core/state"
	t "core/tun"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"net"

	"github.com/metacubex/mihomo/component/dialer"
	"github.com/metacubex/mihomo/component/iface/anet"
	"github.com/metacubex/mihomo/component/process"
	"github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/dns"
	"github.com/metacubex/mihomo/listener/sing_tun"
	"github.com/metacubex/mihomo/log"
	"github.com/metacubex/mihomo/tunnel/statistic"
)

type ProcessMap struct {
	m sync.Map
}

type FdMap struct {
	m sync.Map
}

type FdWaitMap struct {
	m sync.Map
}

func (wm *FdWaitMap) Store(key int64, ch chan struct{}) {
	wm.m.Store(key, ch)
}

func (wm *FdWaitMap) Load(key int64) (chan struct{}, bool) {
	value, ok := wm.m.Load(key)
	if !ok || value == nil {
		return nil, false
	}
	return value.(chan struct{}), true
}

func (wm *FdWaitMap) Delete(key int64) {
	wm.m.Delete(key)
}

type Fd struct {
	Id    int64 `json:"id"`
	Value int64 `json:"value"`
}

var (
	tunListener   *sing_tun.Listener
	fdMap         FdMap
	fdWaitMap     FdWaitMap
	fdCounter     int64 = 0
	counter       int64 = 0
	processMap    ProcessMap
	tunLock       sync.Mutex
	runTime       *time.Time
	errBlocked    = errors.New("blocked")
	keepaliveStop chan struct{}
	keepaliveOnce sync.Once
)

func (cm *ProcessMap) Store(key int64, value string) {
	cm.m.Store(key, value)
}

func (cm *ProcessMap) Load(key int64) (string, bool) {
	value, ok := cm.m.Load(key)
	if !ok || value == nil {
		return "", false
	}
	return value.(string), true
}

func (cm *FdMap) Store(key int64) {
	cm.m.Store(key, struct{}{})
}

func (cm *FdMap) Load(key int64) bool {
	_, ok := cm.m.Load(key)
	return ok
}

func StartTUN(fd int, markSocket func(Fd)) {
	if fd == 0 {
		tunLock.Lock()
		defer tunLock.Unlock()
		now := time.Now()
		runTime = &now
		return
	}
	initSocketHook(markSocket)
	go func() {
		tunLock.Lock()
		defer tunLock.Unlock()
		f := int(fd)
		tunListener, _ = t.Start(f, currentConfig.General.Tun.Device, currentConfig.General.Tun.Stack, currentConfig.General.Tun.DNSHijack)
		if tunListener != nil {
			log.Infoln("TUN address: %v", tunListener.Address())
		}
		now := time.Now()
		runTime = &now
	}()
	// 启动后台保活 goroutine：每 60s 关闭空闲连接防止 NAT 超时
	startKeepalive()
}

func startKeepalive() {
	keepaliveOnce.Do(func() {
		keepaliveStop = make(chan struct{})
	})
	// 先停止已有保活
	stopKeepalive()
	keepaliveStop = make(chan struct{})
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		log.Infoln("[Keepalive] TUN 保活 goroutine 已启动")
		for {
			select {
			case <-ticker.C:
				// 直连模式下不需要健康检查和空闲连接清理
				if currentConfig != nil && string(currentConfig.General.Mode) == "direct" {
					continue
				}
				// 仅在有活跃连接时才做健康检查, 无流量时跳过以降低功耗
				connSnapshot := statistic.DefaultManager.Snapshot()
				if connSnapshot != nil && len(connSnapshot.Connections) > 0 {
					go handleHealthCheckAll()
				}
				func() {
					runLock.Lock()
					defer runLock.Unlock()
					if tunListener == nil {
						return
					}
					// 关闭所有空闲连接，强制 NAT 重新建立映射
					n := 0
					statistic.DefaultManager.Range(func(c statistic.Tracker) bool {
						// 仅关闭已空闲超过 120s 的连接
						if time.Since(c.LastActivity()) > 120*time.Second {
							_ = c.Close()
							n++
						}
						return true
					})
					if n > 0 {
						log.Infoln("[Keepalive] 已关闭 %d 个空闲连接", n)
					}
				}()
			case <-keepaliveStop:
				log.Infoln("[Keepalive] TUN 保活 goroutine 已停止")
				return
			}
		}
	}()
}

func stopKeepalive() {
	if keepaliveStop != nil {
		select {
		case <-keepaliveStop:
			// 已关闭
		default:
			close(keepaliveStop)
		}
	}
	keepaliveOnce = sync.Once{}
}

func GetRunTime() string {
	if runTime == nil {
		return "clash服务未启动"
	}
	return strconv.FormatInt(runTime.UnixMilli(), 10)
}
func ConfigInited() string {
	if currentConfig != nil {
		return "true"
	}
	return "false"
}

func StopTun() {
	stopKeepalive()
	go func() {
		tunLock.Lock()
		defer tunLock.Unlock()

		runTime = nil

		if tunListener != nil {
			_ = tunListener.Close()
		}
		removeSocketHook()
		// 关闭 VPN 时立即刷新 DNS 缓存，避免 HarmonyOS 系统 10 分钟 DNS 缓存 TTL
		// 导致应用继续使用 VPN DNS 解析的过期记录
		dns.FlushCacheWithDefaultResolver()
	}()
}

func SetFdMap(fd C.long) {
	fdInt := int64(fd)
	go func() {
		fdMap.Store(fdInt)
		// 通知等待的 initSocketHook 协程，避免忙等待轮询
		if ch, ok := fdWaitMap.Load(fdInt); ok {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}()
}

func initSocketHook(markSocket func(Fd)) {
	dialer.DefaultSocketHook = func(network, address string, conn syscall.RawConn) error {
		if platform.ShouldBlockConnection() {
			return errBlocked
		}
		return conn.Control(func(fd uintptr) {
			fdInt := int64(fd)
			id := atomic.AddInt64(&fdCounter, 1)

			// 直连模式下无需保护 socket，直接返回
			if currentConfig != nil && string(currentConfig.General.Mode) == "direct" {
				return
			}

			// 创建等待 channel，替代忙等待轮询
			waitCh := make(chan struct{}, 1)
			fdWaitMap.Store(id, waitCh)

			markSocket(Fd{
				Id:    id,
				Value: fdInt,
			})

			// 等待 SetFdMap 通知。
			// 后台无长时任务时，VPN 进程 JS 事件循环会被系统限流，protect IPC 往返
			// 可能超过 500ms；超时会导致出站 fd 未绕过 TUN → 流量回环 → DNS 全断。
			// 放宽到 5s，并记录超时点便于排查（hilog | grep SocketHook protect）
			select {
			case <-waitCh:
			case <-time.After(5 * time.Second):
				log.Warnln("[SocketHook] protect wait timeout, fd=%d id=%d socket un-protected, may loop into TUN", fdInt, id)
			}
			fdWaitMap.Delete(id)
		})
	}
}

func removeSocketHook() {
	dialer.DefaultSocketHook = nil
}

func init() {
	process.DefaultPackageNameResolver = func(metadata *constant.Metadata) (string, error) {
		if metadata == nil {
			return "", process.ErrInvalidNetwork
		}
		id := atomic.AddInt64(&counter, 1)

		timeout := time.After(200 * time.Millisecond)

		// SendMessage(Message{
		// 	Type: ProcessMessage,
		// 	Data: Process{
		// 		Id:       id,
		// 		Metadata: metadata,
		// 	},
		// })

		for {
			select {
			case <-timeout:
				return "", errors.New("package resolver timeout")
			default:
				value, exists := processMap.Load(id)
				if exists {
					return value, nil
				}
				time.Sleep(20 * time.Millisecond)
			}
		}
	}
}

func SetProcessMap(s string) string {
	paramsString := s
	go func() {
		var processMapItem = &ProcessMapItem{}
		err := json.Unmarshal([]byte(paramsString), processMapItem)
		if err == nil {
			processMap.Store(processMapItem.Id, processMapItem.Value)
		}
	}()
	return ""
}

func GetCurrentProfileName() string {
	if state.CurrentState == nil {
		return ""
	}
	return state.CurrentState.CurrentProfileName
}

func GetVpnOptions() string {
	tunLock.Lock()
	defer tunLock.Unlock()
	port := 7980
	if currentConfig != nil {
		port = currentConfig.General.MixedPort
	}
	options := state.AndroidVpnOptions{
		Enable:           state.CurrentState.Enable,
		Port:             port,
		Ipv4Address:      state.CurrentState.TunIp,
		Ipv6Address:      state.GetIpv6Address(),
		AccessControl:    state.CurrentState.AccessControl,
		SystemProxy:      state.CurrentState.SystemProxy,
		AllowBypass:      state.CurrentState.AllowBypass,
		RouteAddress:     state.CurrentState.RouteAddress,
		BypassDomain:     state.CurrentState.BypassDomain,
		DnsServerAddress: state.GetDnsServerAddress(),
		Mtu:              state.CurrentState.Mtu,
	}
	data, err := json.Marshal(options)
	if err != nil {
		fmt.Println("Error:", err)
		return ""
	}
	return string(data)
}

func SetState(s *C.char) {
	paramsString := C.GoString(s)
	err := json.Unmarshal([]byte(paramsString), state.CurrentState)
	if err != nil {
		return
	}
}

func UpdateDns(s *C.char) {
	dnsList := C.GoString(s)
	go func() {
		log.Infoln("[DNS] updateDns %s", dnsList)
		dns.UpdateSystemDNS(strings.Split(dnsList, ","))
		dns.FlushCacheWithDefaultResolver()
	}()
}

func UpdateSystemDns(dnsList string) error {
	log.Infoln("[DNS] updateDns %s", dnsList)
	go func() {
		log.Infoln("[DNS] updateDns %s", dnsList)
		dns.UpdateSystemDNS(strings.Split(dnsList, ","))
		dns.FlushCacheWithDefaultResolver()
	}()
	return nil
}

// LocalConnectionProperties matches connection.getConnectionProperties.
// getIpNeighTable describes remote neighbors, not local interface addresses.
type LocalConnectionProperties struct {
	InterfaceName string `json:"interfaceName"`
	MTU           int    `json:"mtu"`
	LinkAddresses []struct {
		Address      NetAddress `json:"address"`
		PrefixLength int        `json:"prefixLength"`
	} `json:"linkAddresses"`
}
type NetAddress struct {
	Address string `json:"address"`
	Family  int    `json:"family"`
	Port    int    `json:"port"`
}

func SetInterfaces(paramsString string) error {
	var links []LocalConnectionProperties
	if err := json.Unmarshal([]byte(paramsString), &links); err != nil {
		return err
	}
	var interfaces []net.Interface
	addresses := make(map[string][]net.Addr)
	for _, link := range links {
		if link.InterfaceName == "" {
			continue
		}
		name := C.CString(link.InterfaceName)
		index := int(C.if_nametoindex(name))
		C.free(unsafe.Pointer(name))
		if index == 0 {
			return fmt.Errorf("HarmonyOS interface index unavailable: %s", link.InterfaceName)
		}
		flags := net.FlagUp | net.FlagRunning | net.FlagMulticast
		if link.InterfaceName == "lo" {
			flags |= net.FlagLoopback
		}
		interfaces = append(interfaces, net.Interface{Index: index, Name: link.InterfaceName, MTU: link.MTU, Flags: flags})
		for _, addr := range link.LinkAddresses {
			ip := net.ParseIP(addr.Address.Address)
			if ip == nil {
				continue
			}
			bits := 128
			if ip.To4() != nil {
				ip = ip.To4()
				bits = 32
			}
			if addr.PrefixLength < 0 || addr.PrefixLength > bits {
				continue
			}
			addresses[link.InterfaceName] = append(addresses[link.InterfaceName], &net.IPNet{IP: ip, Mask: net.CIDRMask(addr.PrefixLength, bits)})
		}
	}
	if len(interfaces) == 0 {
		return errors.New("HarmonyOS returned no local network interfaces")
	}
	anet.SetSystemInterfaces(interfaces, addresses)
	return nil
}
