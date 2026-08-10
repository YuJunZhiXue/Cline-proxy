# Cline Go Proxy

Cline API 的反向代理服务，支持多账号轮询、OpenAI 和 Anthropic Messages API 双协议、API Key 鉴权，内置中文管理后台。集成 **opencode opencode 免费模型** 统一网关：一个二进制同时服务 Cline 账号池与 zen free 模型，按 model 自动路由。

## 功能

- **双上游统一网关**：按 model 自动路由 — Cline 账号池（`deepseek/deepseek-v4-flash` 等）与 opencode opencode 免费模型（`deepseek-v4-flash-free`、`nemotron-3-ultra-free` 等，匿名 `public` key）
- **双协议兼容**：同时支持 `/v1/chat/completions`（OpenAI）、`/v1/messages`（Anthropic Messages API）、`/v1/responses`（OpenAI Responses API，Cursor 等客户端直连）
- **zen 模型动态同步**：每 10 分钟自动拉取 `https://opencode.ai/zen/v1/models`，新免费模型自动接入；付费模型显式 400 拒绝
- **官方摘要压缩（opencode 机制移植）**：超限时按官方算法 select 尾部预算 → 锚定摘要模板（Objective/Work State/Next Move/Relevant Files）→ 调 zen 模型生成摘要 → 重组 `[摘要+recent]` 继续会话，增量更新摘要；摘要失败自动降级截断
- **多 IP 轮询出口**：zen 上游支持 http/https/socks5 代理池，round_robin/random/fill 策略，绕过单 IP 匿名额度限制
- **token 统计与日志入库**：每请求 JSONL 落盘（`zen-stats.jsonl`），今日/累计聚合、按模型分布，管理后台实时展示
- **多账号轮询**：自动在多个 Cline 账号间切换负载（支持 `round_robin` / `fill` / `random` 策略）
- **中文管理后台**：浏览器访问 `/admin/` 管理账号、API Key、模型配置、请求头、代理设置；`/admin/` 的「opencode 免费模型」页统一管理 zen 上游、代理池、压缩参数、模型与统计（原独立页已合并）
- **API Key 鉴权**：保护代理端点，支持生成/删除多个 API Key
- **System Prompt 覆盖**：项目目录下放 `override.md` 则自动替换系统提示词，不存在则使用客户端自带
- **账号导入**：支持 OAuth 浏览器登录、手动 Token 输入、批量文件导入
- **持久化存储**：账号和 Key 保存在 `.cline-accounts.json`，zen 配置保存在 `.zen-config.json`
- **账号冷却与自动恢复**：命中 429 `INFERENCE_CAP_ERROR` 时自动解析 "Try again in 17h 59m" 并标记冷却，冷却到期自动恢复
- **本地调用统计**：账号列表明确显示「本地今日/累计调用」
- **多平台 CI/CD**：GitHub Actions 自动构建 6 平台二进制

## 快速开始

### 直接运行

```bash
# 编译并启动（默认监听所有网卡，局域网可访问）
go build -o cline-proxy.exe .
./cline-proxy.exe

# 局域网访问地址：http://<本机局域网IP>:3457/admin/
# 仅允许本机访问时：
./cline-proxy.exe -host 127.0.0.1

# 指定端口
./cline-proxy.exe -port 3457

# 构建 + 启动 + 打开浏览器
go run . -start
```

启动后本机访问 http://127.0.0.1:3457/admin/；局域网设备访问 http://<本机局域网IP>:3457/admin/。

监听所有网卡会开放管理后台给同网设备，建议仅在可信局域网使用，并在系统防火墙中限制 3457 端口。

### Docker 部署

```bash
# 构建并启动
docker compose up -d

# 查看日志
docker compose logs -f

# 停止
docker compose down
```

数据持久化在 `./data/` 目录下，`override.md` 会自动从项目根目录挂载到容器内。

## 使用指南

### 1. 添加 Cline 账号

在管理后台 **账号管理** → **导入账号**，选择以下任一方式：

- **OAuth 浏览器登录**：点击按钮弹出 WorkOS 登录窗口，完成后自动填入
- **手动输入 Token**：输入已有账号的 Access Token
- **批量文件导入**：上传包含账号数据的 JSON 文件

账号列表操作列说明：

- **⚡ 测试**：对账号发起一次真实探测请求。成功则将账号置为活跃（清除冷却/过期状态，相当于升级版重置）；失败则按上游返回的等待时长标记冷却并显示预计恢复时间。
- **↻ 重置**：仅重置该账号的「今日调用」，不影响累计调用、状态和 Token。
- **✕ 删除**：从账号池移除该账号。

### 2. 配置客户端

应用（如 Claude Code、Cline）配置为使用此代理：

**OpenAI 格式（/v1/chat/completions）：**
```
Base URL: http://<本机局域网IP>:3457/v1
API Key:  <在管理后台生成的 Key>
Model:    deepseek/deepseek-v4-flash
```

**Anthropic 格式（/v1/messages）：**
```
Base URL: http://<本机局域网IP>:3457/v1
API Key:  <在管理后台生成的 Key>
Model:    deepseek/deepseek-v4-flash
```

### 3. API Key 管理

在后台 **设置** → **API Keys** 中生成和管理。如果未配置任何 Key，代理允许无鉴权访问。

### 4. System Prompt 覆盖

在项目目录下创建 `override.md`，内容将替换所有客户端请求的系统提示词。删除该文件则使用客户端自带的提示词。

### 5. 请求头配置

后台 **设置** → **请求头** 可编辑转发给上游的自定义请求头（如 `x-client-type: cline-cli`）。

## 可用模型（实测）

### 消耗账户额度

| 模型 ID | 状态 | 说明 |
|---------|:----:|------|
| `deepseek/deepseek-v4-pro` | ✅ 可用 · 消耗额度 | DeepSeek V4 Pro |
| `openai/gpt-4.1-nano` | ✅ 可用 · 消耗额度 | GPT-4.1 Nano |
| `qwen/qwen3-235b-a22b` | ✅ 可用 · 消耗额度 | Qwen3 235B |
| `meta-llama/llama-4-maverick` | ✅ 可用 · 消耗额度 | Llama 4 Maverick |
| `google/gemini-2.5-flash` | ⚠️ 响应为空 · 消耗额度 | API 返回 200 但内容为空 |
| `google/gemini-2.5-pro` | ⚠️ 响应为空 · 消耗额度 | API 返回 200 但内容为空 |

### 官方免费模型

| 模型 ID | 状态 | 说明 |
|---------|:----:|------|
| `deepseek/deepseek-v4-flash` | ✅ 可用 · 不消耗额度 | DeepSeek V4 Flash |
| `poolside/laguna-s-2.1:free` | ✅ 可用 · 不消耗额度 | Poolside Laguna S 2.1 |
| `stepfun/step-3.7-flash` | ✅ 可用 · 不消耗额度 | StepFun 3.7 Flash |

### 需要订阅

| 模型 ID | 状态 | 说明 |
|---------|:----:|------|
| `cline-pass/glm-5.2` | ❌ 403 · 需要订阅 | 需要 `cline-pass` 订阅 |
| `cline-pass/deepseek-v4-flash` | ❌ 403 · 需要订阅 | 需要 `cline-pass` 订阅 |
| `cline-pass/qwen3.7-max` | ❌ 403 · 需要订阅 | 需要 `cline-pass` 订阅 |

可在后台 **设置** → **默认模型** 中修改默认模型。

## CI/CD

Release 版本号以 `v` 开头，从 `v0.0.1` 开始按语义版本递增：首次发布为 `v0.0.1`，后续推送自动发布 `v0.0.2`、`v0.0.3` 等版本。

### 6. opencode 免费模型（统一网关）

无需配置，`cline-proxy.exe` 启动即启用 zen 上游（匿名 key `public`）：

**OpenAI 格式（/v1/chat/completions）：**
```
Base URL: http://<本机局域网IP>:3457/v1
API Key:  <在管理后台生成的 Key>
Model:    deepseek-v4-flash-free   # 或别名 deepseek-v4-flash
```

**Anthropic 格式（/v1/messages）：**
```
Model:    deepseek-v4-flash-free
```

**Responses API（/v1/responses，Cursor 等）：**
```
Model:    deepseek-v4-flash-free
```

可用 opencode 免费模型（`GET /v1/models` 实时列出）：

| 模型 ID | 上下文 | 说明 |
|---------|:----:|------|
| `deepseek-v4-flash-free` | 200K | 别名 `deepseek-v4-flash` |
| `nemotron-3-ultra-free` | 1M | 免费里最大上下文 |
| `north-mini-code-free` | 256K | |
| `mimo-v2.5-free` / `ling-3.0-flash-free` / `laguna-s-2.1-free` / `longcat-2.0-free` / `big-pickle` | 200K | |

超限时自动触发官方摘要压缩（`/admin/` → opencode 免费模型 页可调参数）；付费 zen 模型（如 `glm-5.1`）返回 400 拒绝。

## 7. 项目结构

```
├── main.go               入口与 CLI 参数处理
├── proxy.go              HTTP 服务、API 路由与协议转换
├── models.go             官方免费模型同步（Cline）
├── zen.go                opencode 免费模型上游、三态路由、配置持久化
├── compact.go            opencode 官方摘要压缩机制移植
├── proxy_pool.go         zen 上游多 IP 轮询出口（HTTP/SOCKS5 代理池）
├── stats.go              token 统计与 JSONL 日志入库
├── responses.go          /v1/responses（OpenAI Responses API）转换
├── admin.go              管理后台 REST API
├── admin_zen.go          zen 管理页面与 API
├── admin_html.go         管理后台页面
├── auth.go               WorkOS OAuth 登录与 Token 刷新
├── pool.go               账号池管理与持久化
├── types.go              数据结构定义
├── capture.go            OAuth 信息捕获工具
├── http.go               HTTP 客户端与工具函数
├── Dockerfile            Docker 构建配置
├── docker-compose.yml    Docker Compose 配置
├── go.mod                Go 模块定义
├── .cline-accounts.json  账号池数据
├── .zen-config.json      zen 上游配置（代理池、压缩参数）
├── override.md           可选的系统提示词覆盖文件
└── README.md             项目说明
```

---

感谢 [LINUX DO](https://linux.do) 社区
