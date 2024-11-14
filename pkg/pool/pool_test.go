package pool

import (
	"jd-auto-proxy/pkg/conf"
	"log"
	"testing"
	"time"
)

func TestDynamicPoolCache(t *testing.T) {
	t.Run("代理地址缓存测试", func(t *testing.T) {
		fixedAddrs := []string{"127.0.0.1:8080", "127.0.0.1:8081"}
		p := NewDynamicPool(&conf.Config{
			ProxySources: []*conf.ProxySource{
				{
					TTL:       2 * time.Second,
					Type:      "fixed",
					FixedAddr: fixedAddrs[0:1],
				},
			},
		})
		a1, _ := p.GetAddress()
		p.sources[0].FixedAddr = fixedAddrs[1:2]
		time.Sleep(time.Second)
		a2, _ := p.GetAddress()
		log.Printf("a1: %s, a2: %s", a1, a2)
		if a1 != a2 {
			t.Fatal(a1, a2)
		}
		time.Sleep(time.Second * 2)
		a3, _ := p.GetAddress()
		log.Printf("a1: %s, a3: %s", a1, a3)
		if a1 == a3 {
			t.Fatal(a1, a3)
		}
	})
}
