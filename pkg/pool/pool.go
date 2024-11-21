package pool

import (
	"easy-http-proxy-pool/pkg/conf"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

type Pool interface {
	GetAddress() (string, error)
	DisableAddress(addr string)
}

// DisableableSource 代理源存储
// DisableableSource.Disable 初始禁用时间15秒，不断翻倍, 最长120分钟
type DisableableSource struct {
	conf.ProxySource
	disabledAt     time.Time
	disabledFor    time.Duration
	disabledReason string
}

func (s *DisableableSource) IsDisabled() bool {
	if s.disabledAt.IsZero() {
		return false
	}
	return s.disabledAt.Add(s.disabledFor).After(time.Now())
}

func (s *DisableableSource) Enable() {
	s.disabledAt = time.Time{}
	s.disabledFor = 0
	s.disabledReason = ""
}

func (s *DisableableSource) Disable(reason string) {
	s.disabledAt = time.Now()
	s.disabledReason = reason
	if s.disabledFor == 0 {
		s.disabledFor = 15 * time.Second
	} else {
		s.disabledFor *= 2
	}
	if s.disabledFor > (120 * time.Minute) {
		s.disabledFor = 120 * time.Minute
	}
}

type ExpiringAddr struct {
	addr       string
	expiration time.Time
}

// DynamicPool 动态代理池
type DynamicPool struct {
	sources   []*DisableableSource
	mu        sync.Mutex
	addrStore []*ExpiringAddr
}

func NewDynamicPool(config *conf.Config) *DynamicPool {
	s := make([]*DisableableSource, len(config.ProxySources))
	for i, item := range config.ProxySources {
		s[i] = &DisableableSource{ProxySource: *item}
	}
	return &DynamicPool{addrStore: make([]*ExpiringAddr, 0), sources: s}
}

func (r *DynamicPool) cacheAddr(addr string, ttl time.Duration) {
	r.addrStore = append(r.addrStore, &ExpiringAddr{
		addr:       addr,
		expiration: time.Now().Add(ttl),
	})
}

func (r *DynamicPool) peekAddr() (string, bool) {
	if len(r.addrStore) == 0 {
		var zero string
		return zero, false
	}
	for i, item := range r.addrStore {
		if item.expiration.After(time.Now()) {
			// 前面的过期了了
			r.addrStore = r.addrStore[i:]
			return item.addr, true
		}
	}
	// 全部过期了
	r.addrStore = make([]*ExpiringAddr, 0)
	var zero string
	return zero, false
}

// peekSource 寻找一个可用的代理源
func (r *DynamicPool) peekSource() (*DisableableSource, bool) {
	for _, item := range r.sources {
		if !item.IsDisabled() {
			return item, true
		}
	}
	return nil, false
}

// fetchAddress 从指定源中加载一个地址
func (r *DynamicPool) fetchAddress(source *conf.ProxySource) ([]string, error) {
	var loader Loader
	switch source.Type {
	case "fixed":
		loader = &FixedIpLoader{IPs: source.FixedAddr}
	case "common":
	case "":
		loader = &CommonIpLoader{FetchURL: source.FetchURL}
	}
	if loader == nil {
		return nil, fmt.Errorf("unknown source type: %s", source.Type)
	}
	return loader.GetAddress()
}

func (r *DynamicPool) GetAddress() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if peek, ok := r.peekAddr(); ok {
		return peek, nil
	}
	s, ok := r.peekSource()
	if !ok {
		return "", fmt.Errorf("无可用代理源")
	}
	ips, err := r.fetchAddress(&s.ProxySource)
	if err != nil {
		// 禁用
		s.Disable(err.Error())
		slog.Warn("代理源已禁用",
			slog.String("source", s.Name),
			slog.String("reason", s.disabledReason),
			slog.Duration("disabledFor", s.disabledFor),
		)
		return "", err
	}
	for _, addr := range ips {
		r.cacheAddr(addr, s.TTL)
		slog.Debug(fmt.Sprintf("提取代理地址 %s", addr), slog.String("source", s.Name))
	}
	return ips[0], nil
}

// DisableAddress 禁用指定的地址
func (r *DynamicPool) DisableAddress(addr string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, expiringAddr := range r.addrStore {
		if expiringAddr.addr == addr {
			r.addrStore = append(r.addrStore[:i], r.addrStore[i+1:]...)
			return
		}
	}
}
