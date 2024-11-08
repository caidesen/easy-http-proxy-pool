package pool

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type Loader interface {
	GetAddress() ([]string, error)
}

type CommonIpLoader struct {
	FetchURL string
}

func AddressValidator(addr string) bool {
	// IPv4 地址 + 必须的端口号
	pattern := `^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(:[0-9]{1,5})$`
	re := regexp.MustCompile(pattern)
	return re.MatchString(addr)
}

// SplitIPs 根据 \r\n 或 \n 或 \r 切分 IP 字符串，并检查有效性
func (r *CommonIpLoader) splitIp(ipStr string) ([]string, error) {
	ipStr = strings.ReplaceAll(ipStr, "\r\n", "\n")
	ipStr = strings.ReplaceAll(ipStr, "\r", "\n")
	ips := strings.Split(ipStr, "\n")
	var validIPs []string
	for _, ip := range ips {
		ip = strings.TrimSpace(ip) // 去除前后空白
		if ip == "" {
			continue // 跳过空行
		}
		// 检查 IP 是否有效
		if !AddressValidator(ip) {
			return nil, errors.New("无效的 IP 地址: " + ip)
		}
		validIPs = append(validIPs, ip)
	}

	return validIPs, nil
}

func (r *CommonIpLoader) GetAddress() ([]string, error) {
	resp, err := http.Get(r.FetchURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if (resp.StatusCode / 100) != 2 {
		return nil, fmt.Errorf("fetch ip error: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	ips, err := r.splitIp(string(body))
	if err != nil {
		return nil, fmt.Errorf("fetch ip error: %s", string(body))
	}

	for _, line := range strings.Split(string(body), "\r\n") {
		if line == "" {
			continue
		}
		ips = append(ips, line)
	}
	return ips, nil
}
