package proxy

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"jd-auto-proxy/pkg/conf"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

func copyOrWarn(ctx *ProxyCtx, dst io.Writer, src io.Reader, wg *sync.WaitGroup) {
	if _, err := io.Copy(dst, src); err != nil {
		ctx.Warn(fmt.Sprintf("Error copying to client: %s", err))
	}
	wg.Done()
}

func copyAndClose(ctx *ProxyCtx, dst, src halfClosable) {
	if _, err := io.Copy(dst, src); err != nil {
		ctx.Warn(fmt.Sprintf("Error copying to client: %s", err))
	}
	dst.CloseWrite()
	src.CloseRead()
}

// httpError 错误处理
func httpError(ctx *ProxyCtx, w io.WriteCloser, err error) {
	errStr := fmt.Sprintf("HTTP/1.1 502 Bad Gateway\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(err.Error()), err.Error())
	if _, err := io.WriteString(w, errStr); err != nil {
		ctx.Warn(fmt.Sprintf("Error responding to client: %s", err))
	}
	if err := w.Close(); err != nil {
		ctx.Warn(fmt.Sprintf("Error closing client connection: %s", err))
	}
}

// concat 合并数组
func concat(dest []byte, src []byte) (result []byte) {
	result = make([]byte, len(dest)+len(src))
	//将第一个数组传入result
	copy(result, dest)
	//将第二个数组接在尾部，也就是 len(dest):
	copy(result[len(dest):], src)
	return
}

type halfClosable interface {
	net.Conn
	CloseWrite() error
	CloseRead() error
}

// createHttpConnectBytes 重新构造HTTP CONNECT请求报文
// CONNECT xxx.com:443 HTTP/1.1
// Host: xxx.com:443
// ....
func createHttpConnectBytes(req *http.Request) []byte {
	reqByte := []byte(fmt.Sprintf("%s %s %s\r\n", req.Method, req.Host, req.Proto))
	reqByte = concat(reqByte, []byte(fmt.Sprintf("Host: %s\r\n", req.Host)))
	for k, v := range req.Header {
		reqByte = concat(reqByte, []byte(fmt.Sprintf("%s: %s\r\n", k, v[0])))
	}
	reqByte = concat(reqByte, []byte{13, 10})
	all, err := io.ReadAll(req.Body)
	if err == nil {
		reqByte = concat(reqByte, all)
	}
	return reqByte
}

// checkProxyConnectTunnel 检查代理隧道连接
// tcp 连接成功后，返回 HTTP/1.1 200 Connection established 隧道成功建立
func checkProxyConnectTunnel(conn net.Conn) error {
	var buf [1024]byte
	_, err := conn.Read(buf[:])
	if err != nil {
		return err
	}
	if strings.HasPrefix(string(buf[:]), "HTTP/1.1 200 Connection established") {
		return nil
	}
	if strings.HasPrefix(string(buf[:]), "HTTP/1.0 200 Connection established") {
		return nil
	}
	return fmt.Errorf("响应报文不符合预期: %s", string(bytes.TrimSpace(buf[:])))
}

// tcpConnect 发起tcp连接
func tcpConnect(ctx context.Context, addr string) (net.Conn, error) {
	dialContext := (&net.Dialer{
		Timeout:   4 * time.Second,
		KeepAlive: 4 * time.Second,
	}).DialContext
	return dialContext(ctx, "tcp", addr)
}

// createProxyTunnel 创建代理隧道
func createProxyTunnel(ctx *ProxyCtx, addr string) (net.Conn, error) {
	addr, err := ctx.Pool.GetAddress()
	if err != nil {
		ctx.Warn(fmt.Sprintf("获取代理地址失败: %s", err))
		return nil, err
	}

	targetConn, err := tcpConnect(ctx.Req.Context(), addr)
	if err != nil {
		ctx.Warn(fmt.Sprintf("tcp连接失败 %s: %s", addr, err.Error()))
		return nil, err
	}
	ctx.Debug(fmt.Sprintf("tcp连接成功 %s", addr))
	reqBytes := createHttpConnectBytes(ctx.Req)
	ctx.Debug(fmt.Sprintf("http请求报文 %s", string(reqBytes)))
	_, err = targetConn.Write(reqBytes)
	if err != nil {
		ctx.Warn(fmt.Sprintf("代理隧道建立失败 %s: %s", addr, err.Error()))
		return nil, err
	}

	err = checkProxyConnectTunnel(targetConn)
	if err != nil {
		ctx.Warn(fmt.Sprintf("代理隧道连通性检查未通过 %s: %s", addr, err.Error()))
		return nil, err
	}
	return targetConn, nil
}

// tryCreateProxyTunnel 重试创建代理隧道, 如果失败就移除代理地址
func tryCreateProxyTunnel(ctx *ProxyCtx) (net.Conn, error) {
	addr, err := ctx.Pool.GetAddress()
	if err != nil {
		ctx.Warn(fmt.Sprintf("获取代理地址失败: %s", err))
		return nil, err
	}
	ctx.Debug(fmt.Sprintf("获取代理地址: %s", addr))
	targetConn, err := createProxyTunnel(ctx, addr)
	if err != nil {
		ctx.Pool.DisableAddress(addr)
		ctx.Warn(fmt.Sprintf("代理不可用, 已移除: %s", err))
	}
	return targetConn, err
}

// checkHostnameNeedProxy 检查是否需要代理
func checkHostnameNeedProxy(ctx *ProxyCtx) bool {
	host := ctx.Req.Host
	ctx.Debug(fmt.Sprintf("检查主机名 %s 是否需要代理", host))
	for _, regStr := range ctx.conf.ProxyHost {
		reg := regexp.MustCompile(regStr)
		if reg.MatchString(host) {
			ctx.Debug(fmt.Sprintf("主机名 %s 命中代理规则 %s", host, regStr))
			return true
		}
	}
	ctx.Debug(fmt.Sprintf("主机名 %s 不符合代理规则", host))
	return false
}

// HijackConnectHandle 劫持http连接处理
func HijackConnectHandle(ctx *ProxyCtx, clientConn net.Conn) {
	var targetConn net.Conn
	if checkHostnameNeedProxy(ctx) {
		conn, err := tryCreateProxyTunnel(ctx)
		if err != nil {
			ctx.Warn(fmt.Sprintf("当前远程代理不可用，降级为本地请求"))
		} else {
			targetConn = conn
		}
	}
	if targetConn == nil {
		conn, err := tcpConnect(ctx.Req.Context(), ctx.Req.Host)
		if err != nil {
			httpError(ctx, clientConn, err)
			return
		}
		targetConn = conn
	}
	clientConn.Write([]byte("HTTP/1.1 200 Connection established\r\n\r\n"))
	ctx.Debug("隧道建立, 开始正式传输")
	targetTCP, targetOK := targetConn.(halfClosable)
	proxyClientTCP, clientOK := clientConn.(halfClosable)
	if !targetOK || !clientOK {
		var wg sync.WaitGroup
		wg.Add(2)
		go copyOrWarn(ctx, targetConn, clientConn, &wg)
		go copyOrWarn(ctx, clientConn, targetConn, &wg)
		wg.Wait()
		clientConn.Close()
		targetConn.Close()
		return
	} else {
		go copyAndClose(ctx, targetTCP, proxyClientTCP)
		go copyAndClose(ctx, proxyClientTCP, targetTCP)
	}
}

func createNewReq(r *http.Request) (*http.Request, error) {
	request, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		return nil, err
	}
	request.URL = r.URL
	request.Header = r.Header
	return request, err
}

func doRequest(r *http.Request, tr *http.Transport) (*http.Response, error) {
	client := &http.Client{Timeout: 4 * time.Second, Transport: tr}
	return client.Do(r)
}

func getProxyUrl(ctx *ProxyCtx) (*url.URL, error) {
	addr, err := ctx.Pool.GetAddress()
	ctx.Debug(fmt.Sprintf("获取代理地址: %s", addr))
	if err != nil {
		ctx.Warn(fmt.Sprintf("获取代理地址失败: %s", err))
		return nil, err
	}
	proxyUrl, err := url.Parse("http://" + addr)
	if err != nil {
		return nil, err
	}
	return proxyUrl, nil
}

func HttpRequestHandle(ctx *ProxyCtx, w http.ResponseWriter) {
	r, err := createNewReq(ctx.Req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	// 检查是否需要代理
	if checkHostnameNeedProxy(ctx) {
		proxyUrl, err := getProxyUrl(ctx)
		if err != nil {
			ctx.Warn(fmt.Sprintf("当前远程代理不可用，降级为本地请求: %s", err))
		} else {
			tr.Proxy = http.ProxyURL(proxyUrl)
		}
	}
	if conf.IsDebug {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		ctx.Debug("发起代理请求",
			"method", r.Method,
			"url", r.URL.String(),
			"headers", r.Header,
			"body", string(bodyBytes))
	}
	res, err := doRequest(r, tr)
	if err != nil {
		ctx.Warn(fmt.Sprintf("代理请求失败: %s", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if conf.IsDebug {
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return
		}
		ctx.Debug("代理请求结束",
			"statusCode", res.StatusCode,
			"headers", res.Header,
			"body", string(bodyBytes))
	}
	defer res.Body.Close()
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
}
