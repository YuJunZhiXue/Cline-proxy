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
- **今日/总次数统计**：账号列表显示「今日/总次数」，跨日自动重置今日计数；「↻重置」按钮仅重置今日次数，不影响总次数、状态和 Token
- **多平台 CI/CD**：GitHub Actions 自动构建，推送代码到 main 即触发，生成 Windows (amd64/arm64)、Linux (amd64/arm64)、macOS (amd64/arm64) 共 6 个平台二进制，发布标题格式为 `v年_月日_时分`

## 快速开始

### 直接运行

```bash
# 编译并启动（默认端口 3457）
go build -o cline-proxy.exe .
./cline-proxy.exe

# 指定端口
./cline-proxy.exe -port 3457

# 构建 + 启动 + 打开浏览器
go run . -start
```

启动后访问 http://127.0.0.1:3457/admin/ 进入管理后台。

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
- **↻ 重置**：仅重置该账号的「今日使用次数」，不影响总次数、状态和 Token。
- **✕ 删除**：从账号池移除该账号。

### 2. 配置客户端

应用（如 Claude Code、Cline）配置为使用此代理：

**OpenAI 格式（/v1/chat/completions）：**
```
Base URL: http://127.0.0.1:3457/v1
API Key:  <在管理后台生成的 Key>
Model:    cline-free/glm-5.2
```

**Anthropic 格式（/v1/messages）：**
```
Base URL: http://127.0.0.1:3457/v1
API Key:  <在管理后台生成的 Key>
Model:    cline-free/glm-5.2
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
| `deepseek/deepseek-v4-flash` | ⚠️ 响应为空 · 消耗额度 | API 返回 200 但内容为空 |
| `google/gemini-2.5-flash` | ⚠️ 响应为空 · 消耗额度 | API 返回 200 但内容为空 |
| `google/gemini-2.5-pro` | ⚠️ 响应为空 · 消耗额度 | API 返回 200 但内容为空 |

### 不消耗账户额度

| 模型 ID | 状态 | 说明 |
|---------|:----:|------|
| `cline-free/glm-5.2` | ✅ 可用 · 不消耗额度 | 免费模型，无限使用 |
| `cline-pass/glm-5.2` | ❌ 403 · 不消耗额度 | 需要 `cline-pass` 订阅 |
| `cline-pass/deepseek-v4-flash` | ❌ 403 · 不消耗额度 | 需要 `cline-pass` 订阅 |
| `cline-pass/qwen3.7-max` | ❌ 403 · 不消耗额度 | 需要 `cline-pass` 订阅 |

可在后台 **设置** → **默认模型** 中修改默认模型。

## CI/CD

推送代码到 `main` 分支自动触发 GitHub Actions 工作流：

1. 并行构建 6 个平台二进制（Windows/Linux/macOS × amd64/arm64）
2. 自动创建 Release，标题格式为 `v年_月日_时分`（如 `v2026_0804_1415`）
3. 所有二进制作为 Release Assets 上传

工作流配置位于 [.github/workflows/build.yml](.github/workflows/build.yml)。

## 项目结构

```
├── main.go             入口，CLI 参数处理
├── proxy.go            HTTP 服务，API 路由，协议转换，SSE 流式处理
├── admin.go            管理后台 REST API
├── admin_html.go       管理后台前端 HTML（嵌入 Go 二进制）
├── auth.go             WorkOS OAuth 登录与 Token 刷新
├── pool.go             账号池管理、持久化、策略轮询
├── types.go            数据结构定义
├── capture.go          OAuth 信息捕获工具
├── http.go             HTTP 客户端与工具函数
├── build.ps1           多平台交叉编译脚本
├── rebuild.ps1         重建并重启脚本（开发热更新）
├── Dockerfile          Docker 构建
├── docker-compose.yml  Docker Compose 配置
├── .github/workflows/build.yml  GitHub Actions 多平台构建
└── override.md         可选的系统提示词覆盖文件

---

感谢 [LINUX DO](https://linux.do) 社区
```
