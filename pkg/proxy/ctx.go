package proxy

import (
	"easy-http-proxy-pool/pkg/conf"
	"easy-http-proxy-pool/pkg/middleware"
	"easy-http-proxy-pool/pkg/pool"
	"log/slog"
	"net/http"
)

type ProxyCtx struct {
	// Will contain the client request from the proxy
	Req  *http.Request
	Pool pool.Pool
	conf *conf.Config
}

func (ctx *ProxyCtx) getReqInfo() []interface{} {
	return []interface{}{
		"host", ctx.Req.Host,
		"tranceId", middleware.GetReqID(ctx.Req.Context()),
	}
}

func (ctx *ProxyCtx) Log(level slog.Level, msg string, argv ...interface{}) {
	args := ctx.getReqInfo()
	args = append(args, argv...)
	slog.Log(ctx.Req.Context(), level, msg, args...)
}

func (ctx *ProxyCtx) Debug(msg string, argv ...interface{}) {
	ctx.Log(slog.LevelDebug, msg, argv...)
}

func (ctx *ProxyCtx) Info(msg string, argv ...interface{}) {
	ctx.Log(slog.LevelInfo, msg, argv...)
}

func (ctx *ProxyCtx) Warn(msg string, argv ...interface{}) {
	ctx.Log(slog.LevelWarn, msg, argv...)
}

func (ctx *ProxyCtx) Error(msg string, argv ...interface{}) {
	ctx.Log(slog.LevelError, msg, argv...)
}
