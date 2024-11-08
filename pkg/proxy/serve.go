package proxy

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"jd-auto-proxy/pkg/conf"
	"jd-auto-proxy/pkg/pool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ProxyServer struct {
	pool pool.Pool
	conf *conf.Config
}

func NewProxyServer(config *conf.Config) *ProxyServer {
	return &ProxyServer{
		pool: pool.NewDynamicPool(config),
		conf: config,
	}
}

func (s *ProxyServer) handleConnect(w http.ResponseWriter, r *http.Request) {
	ctx := &ProxyCtx{Req: r, Pool: s.pool, conf: s.conf, TraceId: uuid.New().String()}
	hij, ok := w.(http.Hijacker)
	if !ok {
		panic("httpserver does not support hijacking")
	}
	proxyClient, _, e := hij.Hijack()
	if e != nil {
		http.Error(w, e.Error(), http.StatusInternalServerError)
		ctx.Error(e.Error())
	}
	HijackConnectHandle(ctx, proxyClient)
}

func (s *ProxyServer) httpRequestHandle(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		s.handleConnect(w, r)
	} else {
		// todo: 普通的转发 模式
	}
}

func (s *ProxyServer) Listen(addr string) {
	server := &http.Server{
		Addr:      addr,
		TLSConfig: &tls.Config{InsecureSkipVerify: true},
		Handler:   http.HandlerFunc(s.httpRequestHandle),
	}
	go func() {
		slog.Info(fmt.Sprintf("代理服务启动 %s", addr))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error(fmt.Sprintf("代理服务启动失败: %v", err))
			os.Exit(1)
		}
	}()
	// 处理系统信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	// 创建一个 5 秒的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	slog.Info("正在关闭代理服务...")
	if err := server.Shutdown(ctx); err != nil {
		slog.Error(fmt.Sprintf("Server forced to shutdown: %v\n", err))
	}
}
