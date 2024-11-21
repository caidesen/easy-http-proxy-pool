package main

import (
	"easy-http-proxy-pool/pkg/conf"
	"easy-http-proxy-pool/pkg/logger"
	"easy-http-proxy-pool/pkg/proxy"
	"fmt"
	"io"
	"log/slog"
	"os"
)

func main() {
	conf.AppArgsInit()
	if conf.VersionOut {
		fmt.Println("v0.1.4")
		os.Exit(0)
	}
	level := slog.LevelInfo
	if conf.IsDebug {
		level = slog.LevelDebug
	}
	var logWriter io.Writer = os.Stdout
	if conf.LogEnabled {
		logWriter = &logger.DailyWriter{
			LogDir: conf.LogDirPath,
			MaxAge: 3,
		}
	}
	logHandler := logger.NewQingLongLogHandler(logWriter, &slog.HandlerOptions{
		Level: level,
	})
	slog.SetDefault(slog.New(logHandler))
	proxyConfig, _ := conf.ReadFromFile(conf.ConfigPath)
	server := proxy.NewProxyServer(proxyConfig)
	server.Listen(fmt.Sprintf("%s:%s", conf.Host, conf.Port))
}
