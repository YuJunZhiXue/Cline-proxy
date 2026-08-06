# Cline Go Proxy

Cline API 的反向代理服务，支持多账号轮询、OpenAI 和 Anthropic Messages API 双协议、API Key 鉴权，内置中文管理后台。

## 功能

- **双协议兼容**：同时支持 `/v1/chat/completions`（OpenAI）和 `/v1/messages`（Anthropic Messages API）
- **多账号轮询**：自动在多个 Cline 账号间切换负载（支持 `round_robin` / `fill` / `random` 策略）
- **中文管理后台**：浏览器访问 `/admin/` 即可管理账号、API Key、模型配置、请求头、代理设置
- **API Key 鉴权**：保护代理端点，支持生成/删除多个 API Key
- **System Prompt 覆盖**：项目目录下放 `override.md` 则自动替换系统提示词，不存在则使用客户端自带
- **账号导入**：支持 OAuth 浏览器登录、手动 Token 输入、批量文件导入
- **持久化存储**：账号和 Key 保存在 `.cline-accounts.json`
- **账号冷却与自动恢复**：命中 429 `INFERENCE_CAP_ERROR` 时自动解析 "Try again in 17h 59m" 并标记冷却，冷却到期自动恢复，账号列表显示预计恢复时间
- **账号测试**：后台账号管理提供「⚡测试」按钮，对单个账号发起真实探测请求，验证是否可用；测试成功会清除冷却/过期状态（相当于升级版重置），失败会按上游返回的等待时长标记冷却
- **本地调用统计**：账号列表明确显示「本地今日/累计调用」，统计代理转发成功（上游 HTTP 200）的实际调用；跨日自动重置今日调用；「↻重置」按钮仅重置本地今日调用。该数字不是 Cline 官方免费额度，官方每日额度不公开精确次数。
- **多平台 CI/CD**：GitHub Actions 自动构建，推送代码到 main 即触发，生成 Windows (amd64/arm64)、Linux (amd64/arm64)、macOS (amd64/arm64) 共 6 个平台二进制，版本从 `v0.0.1` 开始，之后每次自动递增补丁版本

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

### 构建脚本

```powershell
# 重建并重启（开发热更新：停进程 → 构建 Win x86 → 启动 → 打开浏览器）
.\rebuild.ps1

# 指定端口
.\rebuild.ps1 -Port 3458

# 只构建不启动
.\rebuild.ps1 -NoStart

# 构建不打开浏览器
.\rebuild.ps1 -NoOpenBrowser

# 多平台交叉编译（输出到 dist/）
.\build.ps1 -Targets "windows/amd64","windows/arm64","linux/amd64","linux/arm64","darwin/amd64","darwin/arm64"
```

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

## 项目结构

```
├── main.go               入口与 CLI 参数处理
├── proxy.go              HTTP 服务、API 路由与协议转换
├── models.go             官方免费模型同步
├── admin.go              管理后台 REST API
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
├── override.md           可选的系统提示词覆盖文件
└── README.md             项目说明
```

---

感谢 [LINUX DO](https://linux.do) 社区
