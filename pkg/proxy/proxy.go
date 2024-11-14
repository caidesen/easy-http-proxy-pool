package proxy

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"jd-auto-proxy/pkg/conf"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

func copyRequest(r *http.Request) (*http.Request, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	//if conf.IsDebug {
	//	headerText, _ := json.Marshal(r.Header)
	//	ctx.Debug("发起代理请求",
	//		slog.String("method", r.Method),
	//		slog.String("url", r.URL.String()),
	//		slog.String("headers", string(headerText)),
	//		slog.String("body", string(bodyBytes)))
	//}
	request, err := http.NewRequest(r.Method, "", r.Body)
	if err != nil {
		return nil, err
	}
	request.URL = r.URL
	request.Header = r.Header
	return request, err
}

func doRequest(r *http.Request, tr *http.Transport) (*http.Response, error) {
	client := &http.Client{Timeout: 10 * time.Second, Transport: tr}
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

func tryUnzip(r io.Reader) []byte {
	gzipReader, err := gzip.NewReader(r)
	if err != nil {
		return nil
	}
	unzipped, err := io.ReadAll(gzipReader)
	if err != nil {
		return nil
	}
	return unzipped
}

// safetyHttpProxyRequest 安全的代理请求, 如果失败将会执行本机重试
func safetyHttpProxyRequest(ctx *ProxyCtx, req *http.Request, proxyURL *url.URL) (*http.Response, error) {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if proxyURL != nil {
		tr.Proxy = http.ProxyURL(proxyURL)
	}
	resp, err := doRequest(req, tr)
	if err != nil && proxyURL != nil {
		ctx.Debug(fmt.Sprintf("代理请求失败: %s", err))
		return safetyHttpProxyRequest(ctx, req, nil)
	}
	return resp, err
}

// safetyLogRequest 安全的打印请求报文
func safetyLogRequest(ctx *ProxyCtx, req *http.Request) {
	if !conf.IsDebug {
		return
	}
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return
	}
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	headerText, err := json.Marshal(ctx.Req.Header)
	if err != nil {
		return
	}
	ctx.Debug("发起代理请求",
		slog.String("method", req.Method),
		slog.String("url", req.URL.String()),
		slog.String("headers", string(headerText)),
		slog.String("body", string(bodyBytes)))
}

// safetyLogResponse 安全的打印响应报文
func safetyLogResponse(ctx *ProxyCtx, res *http.Response) {
	if !conf.IsDebug {
		return
	}
	bodyBytes, err := io.ReadAll(res.Body)
	res.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	var bodyText string
	if res.Header.Get("Content-Encoding") == "gzip" {
		bodyText = string(tryUnzip(bytes.NewBuffer(bodyBytes)))
	} else {
		bodyText = string(bodyBytes)
	}
	headerText, err := json.Marshal(res.Header)
	if err != nil {
		return
	}
	ctx.Debug("代理请求结束",
		slog.Int("statusCode", res.StatusCode),
		slog.String("headers", string(headerText)),
		slog.String("body", bodyText),
	)
}

func HttpRequestHandle(ctx *ProxyCtx, w http.ResponseWriter) {
	req, err := copyRequest(ctx.Req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// 检查是否需要代理
	var proxyUrl *url.URL
	if checkHostnameNeedProxy(ctx) {
		pl, err := getProxyUrl(ctx)
		if err != nil {
			ctx.Debug(fmt.Sprintf("当前远程代理不可用，降级为本地请求: %s", err))
		} else {
			proxyUrl = pl
		}
	}
	safetyLogRequest(ctx, req)
	res, err := safetyHttpProxyRequest(ctx, req, proxyUrl)
	if err != nil {
		ctx.Warn(fmt.Sprintf("请求失败: %s", err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer res.Body.Close()
	safetyLogResponse(ctx, res)
	for k, vv := range res.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
}
