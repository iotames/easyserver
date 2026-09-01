# EasyServer — 项目指南

一个 Go 语言编写的 HTTP/TCP/WebSocket 服务器框架，提供简洁的 API 接口服务、静态资源服务、中间件链、Session/Cookie 管理等功能。

---

## 项目结构

```
easyserver.go          — 根包 facade，类型别名 + 便捷函数
response.go            — 根包便捷响应函数（ResponseJson, ResponseText 等）
easyserver_test.go     — 集成测试（根包视角，httptest）

conf/                  — 环境变量配置（USE_EMBED_FILE, STATIC_DIR）
custom/                — embed 嵌入的 JSON 配置文件（cmdlist.json）
hotswap/               — 外部脚本/文件热加载工具（优先搜索文件目录，fallback 到 embed）
log/                   — slog 日志封装（Debug/Info/Warn/Error，可配输出目标、级别）
response/              — API 响应数据结构（ResponseApiData，code/msg/data JSON）

httpsvr/               — HTTP 服务器核心
  server.go            — EasyServer 主结构体
  server_data.go       — 全局数据存储（SetData/GetData，读写锁保护）
  server_middle.go     — 中间件注册（head/middle/tail 三层）及 AddStatic/SetCORS
  context.go           — Context（Writer, Request, Server, DataFlow）
  ctx_func.go          — 工具函数（isPathExists）
  ctx_cookie.go        — Cookie 操作（SetCookie/GetCookie/RemoveCookie/SetJsonCookie）
  ctx_session.go       — Session 管理（内存存储，可替换为 Redis）
  handler.go           — AddHandler/AddGetHandler/AddPostHandler 快捷路由
  router.go            — Routing 结构体，AppendRouting
  middleware.go        — MiddleHandle 接口 + NewMiddle 通用中间件工厂
  middlerouter.go      — 路由中间件（按 path + method 匹配，不支持 path 参数）
  middlecors.go        — CORS 跨域中间件
  middlestatic.go      — 静态资源服务中间件（防目录遍历）
  wwwroot.go           — WWWROOT 全站兜底目录（SetWWWRoot/SetNotFoundHandler）+ 公共文件服务函数
  header.go            — 全局响应头设置（EasyServer.SetHeader）
  html.go              — HTML 模板渲染（text/template，支持自定义分隔符）
  dataflow.go          — 请求级数据流传递（DataFlow，读写锁保护）

tcpsvr/                — TCP/WebSocket 服务器
  tcpserver.go         — Server 主结构体，Run/HandlerMsg/SendMsg
  iface.go             — IClient 接口定义
  user.go              — User（TCP 连接包装，消息 channel，心跳超时）
  handler.go           — Handler/MainHandler（连接管理、HTTP 升级 WebSocket）
  websocket.go         — WebSocket 帧封解包（支持文本/二进制）
  request.go           — HTTP 请求解析 + WebSocket 握手
  writer.go            — webSocketWriter / rawTCPWriter（io.Writer 实现）

examples/              — 使用示例（apisvr, tcpsvr, middlesvr, staticserver 等）
```

---

## 核心架构

### HTTP 服务器流程

```
EasyServer.ServeHTTP(w, r):
  1. NewDataFlow() — 创建请求级数据流
  2. 按顺序执行 s.middles 切片中的所有中间件（经 writtenWriter 包装跟踪响应是否已写）
  3. 任一中间件返回 false 则中断链
  4. 链尾兜底：仅当链自然走完且未写响应时，依次尝试 WWWROOT 兜底文件、最终 404
```

### WWWROOT 兜底与自定义 404

路由中间件匹配不到时**不再直接写 404**（`return true` 不写响应），由 `ServeHTTP` 链尾统一处理：
先尝试 `SetWWWRoot(dir)` 设置的兜底目录（按请求路径定位文件，目录请求尝试其下 `index.html`），
仍未命中才调用 `SetNotFoundHandler` 注册的自定义 404（默认 `ResponseNotFound` JSON 响应体）。
两者每请求经 `fieldLock` 读取，**随时调用即刻生效，无需重启**。框架自身不读配置文件，
外部配置（如 easyconf）读出后调 API 传入即可（推荐环境变量 KEY：`WWWROOT`）。

**兜底触发条件**：仅当中间件链自然执行完毕（无任何中间件 `return false`）且全程未写响应时才兜底。
链被中间件主动中断时一律不兜底——拦截请求的中间件**必须自行写响应**，未写则维持历史行为
（隐式空 200），不会把 wwwroot 文件提供给被拦截的请求。CORS 预检 OPTIONS（设置响应头后
`return false` 不写体）即走此路径，保持空 200。wwwroot 兜底只应答 GET/HEAD，其他方法直接走 404。

有意的行为变化：路由未命中时 tail 中间件看到的是「未写响应」；路由命中但 handler 未写任何响应时
会走兜底/404（原先为空 200）。因此 handler 若在 goroutine 中异步写响应，必须保证同步返回前已写完，
否则链尾可能抢先写入 404。默认 404 响应体保持现状（JSON、无 HTTP 404 状态码）。

### 中间件链组装时机

中间件在 `listenPrepare()` 中组装，该方法由 `ListenAndServe()` / `ListenAndServeTLS()` 调用；
首个请求经 `ServeHTTP` 的 `sync.Once`（`initOnce`）也会触发，两次入口由幂等守卫（`len(s.middles) > 0`）保证只组装一次。
组装顺序：

```
headMiddles → routerMiddle → tailMiddles
```

欢迎横幅与「routingList 未设置」警告也在此方法首次执行时打印；`SetQuiet(true)`（须在首次监听前调用）可静默两者，
供纯代理/无路由场景的应用（如 rocksys 主引擎）控制启动日志。

**关键陷阱**：`listenPrepare()` **不**在 `ServeHTTP()` 中调用。这意味着通过 `httptest` 直接调用 `s.ServeHTTP(w, req)` 的测试会失败，因为 `s.middles` 为空（路由和 CORS 中间件均未注入）。测试实际上是 `go test` 时就已经注入了的——但当前测试并未调用 `ListenAndServe`。

### TCP/WebSocket 服务器

- `tcpsvr.NewServer(addr, dropAfterSec)` 创建单例（包级 `svr` 变量）
- `Run()` 启动 TCP listener，每个连接 goroutine 处理
- 首条消息检测 HTTP Upgrade → WebSocket 握手
- 之后按 WebSocket 帧协议通信（文本 0x1 / 二进制 0x2）
- 心跳超时机制：`DropAfter` 秒无消息则断开

---

## 关键约定与陷阱

### ServeHTTP 可直接用于 httptest

`EasyServer` 实现了 `http.Handler` 接口。`ServeHTTP` 首次调用经 `initOnce` 触发 `listenPrepare()`，
因此通过 `httptest` 直接调用 `s.ServeHTTP(w, req)` 也能完成中间件链组装，无需 `ListenAndServe`。
测试中建议先 `s.SetQuiet(true)` 关闭欢迎横幅等启动日志。注意 `AddHandler`/`AddMiddleHead` 等
注册动作必须在首次请求前完成（中间件链只组装一次）。

### 路径匹配是精确匹配

路由中间件（`middlerouter.go`）使用 `rt.Path == rpath` 的**精确字符串比较**，不支持路径参数（如 `/user/:id`）或通配符。这是已知功能限制。

### 静态文件防目录遍历

`middlestatic.go` / `wwwroot.go` 使用 `path.Clean`（POSIX 语义）+ `filepath.FromSlash` + `filepath.Join` 防止 `../` 攻击。
**必须用 `path.Clean` 而非 `filepath.Clean`**：Windows 下 URL 以 `/` 开头拼接成 `//` 时，`filepath.Clean`
会将其当作 UNC 前缀保留、`..` 不被折叠，可逃逸目录（已修复的真实漏洞）。必须先 `os.Stat` 检查存在性，再 `os.Open` 打开文件。

### CORS 只能设置一次

`SetCORS` 使用包级 `hasSetCORS` 标志防止重复设置，重复调用返回错误。

### AddStatic 不能重复注册同一路径

`AddStatic` 使用包级 `staticMap` 防止重复注册 URL 前缀，返回 `"already exists"` 错误。

### GlobalData 的 Rewritable 语义

- `SetDataReadonly` 写入后，该 key 不能再修改
- `SetData` 写入后可以覆盖，但 `SetDataReadonly` 不允许覆盖已存在的可写 key（返回错误）
- 同样语义适用于 `DataFlow` 的 `SetData` / `SetDataReadonly`

### 根包 facade

`easyserver.go` 是类型别名和便捷函数的入口包，实际逻辑在 `httpsvr` 和 `response` 子包。新 API 可以加在根包，也可以用 `httpsvr.Context` 等直接访问。

### 日志基于 slog

`log` 子包封装了 `log/slog`，使用 `sync.Once` 惰性初始化。默认级别为 `LevelInfo`，输出到 stdout。可调用 `log.SetLevel`、`log.SetLogWriter` 修改，但修改需在首次调用日志函数前生效。

### Session 默认使用内存存储

`memorySessionStore` 实现，通过 JSON 序列化深拷贝数据。生产环境应通过 `WithSessionStore` 替换为 Redis 等外部存储。

---

## 关键文件定位

| 功能 | 位置 |
|---|---|
| 服务器主入口 | `httpsvr/server.go:12` |
| 中间件定义 | `httpsvr/middleware.go:7` |
| CORS 中间件 | `httpsvr/middlecors.go` |
| 静态资源 | `httpsvr/middlestatic.go` |
| WWWROOT 兜底 / 自定义 404 | `httpsvr/wwwroot.go` |
| 请求上下文 | `httpsvr/context.go:13` |
| Session 管理 | `httpsvr/ctx_session.go` |
| Cookie 管理 | `httpsvr/ctx_cookie.go` |
| 模板渲染 | `httpsvr/html.go` |
| 路由匹配 | `httpsvr/middlerouter.go:18` |
| TCP 连接处理 | `tcpsvr/handler.go:12` |
| WebSocket 封帧 | `tcpsvr/websocket.go` |
| API 响应封装 | `response/apidata.go` |
| 热加载脚本 | `hotswap/script.go` |

---

## 命令

```bash
go build ./...          # 编译所有包
go test -v ./...        # 运行所有测试（含集成测试）
go vet ./...            # 静态检查
```

测试使用 `httptest.NewRecorder` + `httptest.NewRequest`，无需网络端口。`TestMain` 中创建临时静态文件目录并在结束后清理。

---

## 版本

当前版本权威来源为仓库根目录 `version.txt`。
运行时值由 `httpsvr/server.go` 中的`var MAIN_VERSION` 承载（默认 `v1.5.0`，与 version.txt 保持一致）；
构建时可用 `-ldflags "-X github.com/iotames/easyserver/httpsvr.MAIN_VERSION=$(cat version.txt)"` 注入覆盖。
升级版本时只需编辑 `version.txt`，并同步 `httpsvr/server.go` 的默认值。
