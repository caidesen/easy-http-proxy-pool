package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
)

// QingLongLogHandler 模拟qinglong系统日志
type QingLongLogHandler struct {
	slog.Handler
	out io.Writer
}
type QingLongLogHandlerOptions struct {
	SlogOpts slog.HandlerOptions
	out      io.Writer
}

func NewQingLongLogHandler(out io.Writer, opts *slog.HandlerOptions) *QingLongLogHandler {
	return &QingLongLogHandler{
		out:     out,
		Handler: slog.NewTextHandler(out, opts),
	}
}

func (q *QingLongLogHandler) Handle(ctx context.Context, r slog.Record) error {
	levelText := r.Level.String()
	attrText := ""
	r.Attrs(func(a slog.Attr) bool {
		attrText += fmt.Sprintf(" %s=%s", a.Key, a.Value.String())
		return true
	})
	// 年月日
	timeStr := r.Time.Format("2006-01-02 15:04:05")
	fmt.Fprintf(q.out, "[%s] [%s] [proxy] %s%s\n", levelText, timeStr, r.Message, attrText)
	return nil
}
