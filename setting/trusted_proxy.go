package setting

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type staffProxyPolicy struct {
	trusted []netip.Prefix
	only    bool
}

var currentStaffProxyPolicy atomic.Pointer[staffProxyPolicy]

func loadStaffProxyPolicy(config *viper.Viper) error {
	if config.Get("app") == nil {
		return fmt.Errorf("config.yaml 缺少 app 配置段")
	}
	var entries []string
	switch raw := config.Get("app.trusted_proxies").(type) {
	case nil:
		entries = []string{"127.0.0.1", "::1"}
		if legacy := strings.TrimSpace(os.Getenv("GOFLY_TRUSTED_PROXIES")); legacy != "" {
			for _, entry := range strings.Split(legacy, ",") {
				if strings.TrimSpace(entry) != "" {
					entries = append(entries, entry)
				}
			}
		}
	case []string:
		entries = raw
	case []interface{}:
		for _, item := range raw {
			entry, ok := item.(string)
			if !ok {
				return fmt.Errorf("app.trusted_proxies 的每项必须是 IP 或网段字符串")
			}
			entries = append(entries, entry)
		}
	default:
		return fmt.Errorf("app.trusted_proxies 必须是 IP 或网段列表")
	}
	policy := &staffProxyPolicy{}
	if raw := config.Get("app.require_trusted_proxy"); raw != nil {
		only, ok := raw.(bool)
		if !ok {
			return fmt.Errorf("app.require_trusted_proxy 必须是 true 或 false")
		}
		policy.only = only
	}
	for i, entry := range entries {
		entry = strings.TrimSpace(entry)
		if ip, err := netip.ParseAddr(entry); err == nil && ip.Zone() == "" {
			ip = ip.Unmap()
			policy.trusted = append(policy.trusted, netip.PrefixFrom(ip, ip.BitLen()))
			continue
		}
		prefix, err := netip.ParsePrefix(entry)
		if err != nil || prefix.Bits() == 0 || prefix.Addr().Is4In6() {
			return fmt.Errorf("app.trusted_proxies 第 %d 项不是有效的代理 IP 或网段", i+1)
		}
		policy.trusted = append(policy.trusted, prefix.Masked())
	}
	if policy.only && len(policy.trusted) == 0 {
		return fmt.Errorf("启用 app.require_trusted_proxy 时，app.trusted_proxies 不能为空")
	}
	currentStaffProxyPolicy.Store(policy)
	return nil
}

func reloadStaffProxyPolicy(path string) error {
	config := viper.New()
	config.SetConfigFile(path)
	if err := config.ReadInConfig(); err != nil {
		return err
	}
	return loadStaffProxyPolicy(config)
}

// Watch the directory so the rule also reloads when an editor saves by rename.
func watchStaffProxyConfig(path string) (func(), error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(filepath.Dir(path)); err != nil {
		_ = watcher.Close()
		return nil, err
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		var timer *time.Timer
		var pending <-chan time.Time
		defer func() {
			if timer != nil {
				timer.Stop()
			}
		}()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if filepath.Clean(event.Name) != path || event.Op&(fsnotify.Create|fsnotify.Write|fsnotify.Rename|fsnotify.Remove) == 0 {
					continue
				}
				// Let truncate/write and rename/create finish before reading.
				const settleTime = 200 * time.Millisecond
				if timer == nil {
					timer = time.NewTimer(settleTime)
				} else {
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(settleTime)
				}
				pending = timer.C
			case <-pending:
				pending = nil
				if err := reloadStaffProxyPolicy(path); err != nil {
					log.Printf("代理信任配置未更新: %v", err)
					continue
				}
				log.Print("代理信任配置已重新加载")
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("代理信任配置监听异常: %v", err)
			}
		}
	}()
	return func() {
		_ = watcher.Close()
		<-done
	}, nil
}

func (p *staffProxyPolicy) trusts(remoteAddr string) bool {
	peer, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return false
	}
	addr, err := netip.ParseAddr(peer)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	if p == nil {
		return addr.IsLoopback()
	}
	for _, network := range p.trusted {
		if network.Contains(addr) {
			return true
		}
	}
	return false
}

func IsTrustedProxy(remoteAddr string) bool {
	return currentStaffProxyPolicy.Load().trusts(remoteAddr)
}

func StaffProxyAllowed(r *http.Request) bool {
	policy := currentStaffProxyPolicy.Load()
	if policy.trusts(r.RemoteAddr) {
		return true
	}
	if policy != nil && policy.only {
		return false
	}
	for _, header := range []string{"Forwarded", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Forwarded-For", "X-Real-IP"} {
		if len(r.Header.Values(header)) > 0 {
			return false
		}
	}
	return true
}
