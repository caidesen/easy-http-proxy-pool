package logger

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

type DailyWriter struct {
	LogDir  string
	MaxAge  int
	file    *os.File
	current time.Time
	mu      sync.Mutex
}

func (d *DailyWriter) Write(b []byte) (n int, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.checkRotate(); err != nil {
		return 0, err
	}
	return d.file.Write(b)
}

func (d *DailyWriter) checkRotate() error {
	today := time.Now().Truncate(24 * time.Hour)
	if !d.current.Equal(today) {
		if err := d.rotate(); err != nil {
			return err
		}
	}
	return nil
}
func isValidLogFileName(fileName string) bool {
	// 正则表达式匹配 proxy.YYYY-MM-DD.log 格式
	regex := `^proxy\.\d{4}-\d{2}-\d{2}\.log$`
	matched, err := regexp.MatchString(regex, fileName)
	if err != nil {
		fmt.Println("正则表达式错误:", err)
		return false
	}
	return matched
}

func (d *DailyWriter) removeFile() error {
	now := time.Now()
	threshold := now.Add(-time.Duration(d.MaxAge) * 24 * time.Hour)
	// 遍历日志文件夹
	return filepath.Walk(d.LogDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 检查是否是文件
		if !info.IsDir() && isValidLogFileName(info.Name()) {
			if info.ModTime().Before(threshold) {
				fmt.Printf("删除文件: %s\n", path)
				err := os.Remove(path)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (d *DailyWriter) rotate() error {
	if d.file != nil {
		d.file.Close()
	}

	today := time.Now().Format("2006-01-02")
	filename := path.Join(d.LogDir, "proxy."+today+".log")
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	d.file = file
	d.current = time.Now().Truncate(24 * time.Hour)
	return d.removeFile()
}

func (d *DailyWriter) Close() error {
	if d.file != nil {
		return d.file.Close()
	}
	return nil
}
