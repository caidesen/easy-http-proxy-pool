package main

import (
	"fmt"
	"io"
	"jd-auto-proxy/pkg/conf"
	"jd-auto-proxy/pkg/logger"
	"jd-auto-proxy/pkg/proxy"
	"log/slog"
	"os"
)

func main() {
	if conf.VersionOut {
		fmt.Println("v0.0.1")
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
