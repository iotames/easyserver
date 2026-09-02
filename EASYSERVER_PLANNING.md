# EASYSERVER_PLANNING.md

# EasyServer 规划

## 1. 定位

EasyServer 不以"另一个 Gin"为目标，定位为：

> **轻量、高性能、可扩展的 Go 网络服务运行时/流量处理底座。**

职责链：`HTTP / TCP / WebSocket → EasyServer（Context / Pipeline）→ Middleware / Handler → 业务或上层系统`。

RockSys 是 EasyServer 的重要上层使用者。EasyServer 优先解决性能、并发、Pipeline、Context、生命周期、可扩展性，而不是堆积 Web Framework 功能。

---

## 2. 现状基线

| 能力 | 现状 |
|---|---|
| HTTP / TCP Server（`httpsvr`、`tcpsvr`）、response 封装 | ✅ 已完成 |
| WWWROOT 兜底静态目录、自定义 404 | ✅ 已完成 |
| 生命周期统一（Start / Shutdown / Graceful） | 🔶 部分（HTTP/TCP 已有，未覆盖 WorkerPool 等组件） |
| Server 保护参数（ReadTimeout / WriteTimeout 等） | 🔶 部分（逐项核对补齐） |
| Context 统一 / Context Pool | ⬜ 未开始（当前 Context / Pipeline / DataFlow 均在 RockSys 侧 `internal/`） |
| WorkerPool（Runtime 级 Executor） | 🔶 代码在 RockSys 仓库 `internal/workpool`（另有游离 `internal/workpool.go` 待整理），未下沉 |
| Metrics / pprof | ⬜ 未开始 |

> **关键认知**：现状的 chain / dataflow / engine / workpool 都在 RockSys 仓库。短期规划的主体工程是一次**能力下沉迁移**，不只是原地优化。

---

## 3. 短期规划

### P0-1 能力下沉（立项，迁移优先于优化）

将 RockSys 侧与流量处理底座强相关的能力迁入 easyserver：

- `Context / Pipeline / DataFlow`（来自 `internal/chain`、`internal/dataflow`、`internal/engine`）；
- `workpool`（含清理游离的 `internal/workpool.go`）。

验收：迁移后 RockSys 仅保留业务插件层引用 easyserver；easyserver 自身测试覆盖迁移能力。

### P0-2 HTTP 热路径优化

沿 `Request → Context → Middleware → DataFlow → Handler` 主线，降低 allocation、GC 压力与不必要的锁。
**验收口径**：先建立基准 benchmark（`go test -bench` + pprof）作为基线，以 allocs/op、ns/op、P99 变化为依据，不做猜测式优化。

### P0-3 Context 统一与 Pool

统一 Context 承载 Request / Response / DataFlow / Server / Route / 生命周期，`sync.Pool` 复用（Acquire → Process → Reset → Release）。**重点确保 Reset 完整，避免请求数据残留**（需附带残留检测测试）。

### P0-4 DataFlow 优化

保留现有设计，benchmark + 锁竞争 + allocation 分析后优化 Get/Set 热路径。**不为了理论性能推翻现有设计。**

### P0-5 并发能力

- **WorkerPool 纳入 Runtime**：作为 Executor 服务异步 / CPU 密集 / 审计 / 日志 / 统计 / 后台任务，**不默认接管普通 HTTP 请求**；
- **Admission Control**：全局 / 任务 / 组件三级并发控制，为限流、背压、过载保护留接口；
- **Backpressure**：基于有界队列 + 有限 Worker，过载时拒绝 / 超时 / 降级，避免任务无限堆积。

### P1 生产能力

- **Server 保护参数**：ReadTimeout / ReadHeaderTimeout / WriteTimeout / IdleTimeout / MaxHeaderBytes / MaxBodySize 逐项核对补齐（面向网关高并发场景）；
- **Lifecycle**：统一 Start / Shutdown / Graceful / Drain / Stop，覆盖 HTTP、TCP、WorkerPool；
- **Metrics / pprof**：Request / Latency / Error / Goroutine / Worker / Queue / CPU / Memory / GC 基础观测 + pprof 端点。

---

## 4. 中长期规划

- **Pipeline Runtime**：Pipeline 成为最核心抽象（Middleware / Security / Routing / Processing / Handler）。
- **Executor 抽象**：`Executor` 接口（Submit / TrySubmit），派生 Inline / WorkerPool / Future 等实现，不把 WorkerPool 直接暴露给上层。
- **Router 演进**：二期后视路由规模升级为 Method → Radix Tree → RouteMatch。
- **Multi-Transport**：统一 Transport（HTTP / TCP / WebSocket / QUIC），复用 Context / Lifecycle / Executor / Pipeline。
- **Runtime 化**：最终形成 Transport + Context + Pipeline + Executor + Admission + Backpressure + Lifecycle + Observability 的完整运行时。

---

## 5. 短期明确不做

- **Router Tree**：当前路由规模不是主要矛盾，保留现有实现，二期再议。
- **Web Framework 大而全**：不做 Template、ORM、Binding 体系、MVC。
- **自研 HTTP 协议栈**：继续基于 `net/http`。

---

## 6. 核心原则

1. 不重复造 net/http
2. 性能优先，但以 benchmark 为依据
3. WorkerPool 用于受控任务，不强行接管 HTTP 请求
4. Router 暂缓，二期再做
5. DataFlow 不过早重构
6. 保持 API 简洁
7. 优先建设 Runtime 能力，而不是 Web Framework 功能

最终目标：**轻、快、稳、可扩展**的网络运行时，服务 RockSys 及其他网络服务。
