# BiliProxyM3U8 / BProxy

B站视频反向代理服务器，用于在本地播放器中播放B站视频

- 生成 M3U8 播放列表和 DASH MPD 清单
- 支持多分P视频
- 可选择视频编码（AV1/HEVC/AVC）和画质
- 通过代理转发视频流，支持 Range 请求
- 支持扫码登录（大会员画质需要）

## 已知问题

- PotPlayer 播放 DASH 流时可能在播放一段时间后停止 (PotPlayer不主动触发对剩余段的缓冲)，需要手动 seek 继续
- PotPlayer 每次 seek 都会 GET 一次，即使 seek 的范围很小
- B站某些 CDN 镜像的 TLS 证书配置有问题，需要使用 `-insecure` 跳过验证

## 使用

### 启动服务

```bash
./BiliProxyM3U8
```

默认监听 `:2233`，然后在播放器中打开：

```plaintext
http://localhost:2233/v1/video/{av或BV号}
```

### 命令行参数

```plaintext
-listen string
    服务器监听地址 (默认 ":2233")
    示例: -listen :8080, -listen 127.0.0.1:2233

-codec string
    编码优先级，逗号分隔 (默认 "hevc,avc,av1")
    支持: av1/av01, hevc/h265/h.265, avc/h264/h.264

-quality string
    最高画质 (默认 "1080P")
    可选: 8K, DOLBY, HDR, 4K, 1080P60, 1080P+, 1080P, 720P60, 720P, 480P, 360P, 240P

-proxy
    是否使用 HTTP_PROXY 环境变量 (默认 true)
    禁用: -proxy=false

-insecure
    跳过 TLS 证书验证 (某些 CDN 镜像需要)

-login
    仅执行登录后退出

-debug
    启用 debug 日志

-trace
    启用 trace 日志
```

### 示例

```bash
# 优先 av1, 优先 4K
./BiliProxyM3U8 -codec "av1,hevc,avc" -quality 4K

# 跳过证书验证（某些 CDN 需要）
./BiliProxyM3U8 -insecure

# 自定义监听地址
./BiliProxyM3U8 -listen 0.0.0.0:8080

# 启用 debug 日志
./BiliProxyM3U8 -debug
```

### 登录

不登录最高 1080P，更高画质需要登录：

```bash
./BiliProxyM3U8 -login
```

扫码登录后，凭据会保存到 `bilibili_identity`

## API 端点

### `/v1/video/{id}`

- 无 `p` 参数：返回 M3U8 播放列表（多分P视频）
- 有 `p` 参数：返回指定分P的 MPD 描述文件

示例：

```plaintext
http://localhost:2233/v1/video/av116055351558851
http://localhost:2233/v1/video/BV1F9chzrEwq?p=1
```

### `/v1/proxy`

反代B站视频直链

## License

MIT
