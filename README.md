# easy-http-proxy-pool

## 描述
功能非常简单的代理池工具。

找了一圈相关工具，要么不开源，要么功能复杂，遂自己写了个，仅供学习使用。

## 使用方法

### docker run
```shell
docker run -d \
  --name http-proxy-pool \
  --restart unless-stopped \
  -p 8001:8001 \
  -v ./config:/etc/proxy-pool \
  asen001/easy-http-proxy-pool:latest
```

### docker compose
```yaml
services:
  proxy-pool:
    image: asen001/easy-http-proxy-pool:latest
    volumes:
      - ./config:/etc/proxy-pool
    ports:
      - 8001:8001
    restart: unless-stopped
    container_name: easy-http-proxy-pool
    environment:
      - PROXY_SERVER_DEBUG=false
```

### 配置

第一次运行时会创建配置文件，修改一下就能用了
```yaml
host: # 匹配上的主机名才会使用远程代理
  - .+\.baidu\.com     # 正则表达式
  - .+\.xxxx\.com      # 可以配置多个
sources:
  - name: 携趣
    fetchURL: http://api.xiequ.cn/VAD/GetIp.aspx?xxxxx=xxxxx # 提取地址
    ttl: 27s # 代理过期时间
```