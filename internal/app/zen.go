package app

import (
	"cline-go-proxy/internal/kit"
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// ZenModel opencode zen 免费模型定义
type ZenModel struct {
	ID      string   `json:"id"`
	Aliases []string `json:"aliases,omitempty"`
	Context int      `json:"context"`
	Output  int      `json:"output"`
	Source  string   `json:"source"` // seed=内置 / synced=动态同步
}

var zenSeedModels = []ZenModel{
	{"deepseek-v4-flash-free", []string{"deepseek-v4-flash", "deepseek-v4"}, 200000, 128000, "seed"},
	{"mimo-v2.5-free", []string{"mimo-v2.5", "mimo"}, 200000, 32000, "seed"},
	{"ling-3.0-flash-free", []string{"ling-3.0-flash", "ling"}, 200000, 32768, "seed"},
	{"nemotron-3-ultra-free", []string{"nemotron-3-ultra", "nemotron"}, 1000000, 128000, "seed"},
	{"north-mini-code-free", []string{"north-mini-code", "north-mini"}, 256000, 64000, "seed"},
	{"laguna-s-2.1-free", []string{"laguna-s-2.1", "laguna"}, 200000, 32768, "seed"},
	{"longcat-2.0-free", []string{"longcat-2.0", "longcat"}, 200000, 32768, "seed"},
	{"big-pickle", nil, 200000, 32000, "seed"},
}

var (
	zenModelsMu sync.RWMutex
	zenModels   = make(map[string]*ZenModel) // 主表:ID
	zenAliases  = make(map[string]*ZenModel) // 别名表
)

const zenAPIBase = "https://opencode.ai/zen/v1"

func initZenModels() {
	zenModelsMu.Lock()
	defer zenModelsMu.Unlock()
	if len(zenModels) > 0 {
		return
	}
	for _, m := range zenSeedModels {
		cp := m
		zenModels[cp.ID] = &cp
		for _, a := range cp.Aliases {
			zenAliases[a] = &cp
		}
	}
}

// resolveZenModel 解析模型名到 zen 模型。支持 "opencode/<id>" 前缀与别名。
// 别名优先: 同步来的付费同名模型(如 deepseek-v4-flash)不会覆盖 free 别名解析。
func resolveZenModel(id string) (*ZenModel, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, false
	}
	zenModelsMu.RLock()
	defer zenModelsMu.RUnlock()
	if m, ok := zenAliases[id]; ok {
		return m, true
	}
	if strings.HasPrefix(id, "opencode/") {
		short := strings.TrimPrefix(id, "opencode/")
		if m, ok := zenAliases[short]; ok {
			return m, true
		}
		if m, ok := zenModels[short]; ok {
			return m, true
		}
	}
	if m, ok := zenModels[id]; ok {
		return m, true
	}
	return nil, false
}

// isZenFreeModel 免费判定: seed 白名单 或 ID 带 -free 后缀
func isZenFreeModel(m *ZenModel) bool {
	if m == nil {
		return false
	}
	return m.Source == "seed" || strings.HasSuffix(m.ID, "-free")
}

// resolveZenFreeModel 只解析免费 zen 模型
func resolveZenFreeModel(id string) (*ZenModel, bool) {
	m, ok := resolveZenModel(id)
	if !ok || !isZenFreeModel(m) {
		return nil, false
	}
	return m, true
}

// routeModel 决定请求走哪个上游: "zen" / "cline" / "reject"
// zen 免费模型 -> zen; zen 付费模型 -> reject(400); 其他 -> cline
// 故障转移: zen 连续失败期间,zen 免费模型请求临时路由到 cline 账号池
func routeModel(id string) string {
	id = strings.TrimSpace(id)
	initZenModels()
	cfg := getZenConfig()
	if zm, ok := resolveZenModel(id); ok {
		if isZenFreeModel(zm) {
			// 与 cline 模型表冲突时(几乎不可能)走 cline
			initModelsCache()
			modelsMu.Lock()
			_, inCline := modelsCache[id]
			modelsMu.Unlock()
			if !inCline {
				if cfg.Failover && zenFailedNow() {
					log.Printf("  failover: zen degraded, %q routed to cline pool", id)
					return "cline"
				}
				return "zen"
			}
		} else {
			return "reject"
		}
	}
	if strings.HasPrefix(id, "opencode/") {
		short := strings.TrimPrefix(id, "opencode/")
		if zm, ok := resolveZenModel(short); ok {
			if isZenFreeModel(zm) {
				if cfg.Failover && zenFailedNow() {
					log.Printf("  failover: zen degraded, %q routed to cline pool", id)
					return "cline"
				}
				return "zen"
			}
			return "reject"
		}
	}
	return "cline"
}

// ============ zen 配置 ============

type zenCompactConfig struct {
	Auto         bool   `json:"auto"`         // 官方风格摘要压缩开关
	Buffer       int    `json:"buffer"`       // 预留输出缓冲 token,默认 20000
	KeepTokens   int    `json:"keepTokens"`   // 尾部保留 token 预算,默认 8000
	SummaryModel string `json:"summaryModel"` // 摘要模型,空=用请求模型
	MaxSummary   int    `json:"maxSummary"`   // 摘要最大输出 token,默认 4096
}

type zenConfigData struct {
	Enabled         bool             `json:"enabled"`
	Key             string           `json:"key"`
	BaseURL         string           `json:"baseURL"`
	Proxies         []string         `json:"proxies"`         // http(s)/socks5 代理,轮询出口
	ProxyStrategy   string           `json:"proxyStrategy"`   // round_robin / random / fill
	MaxConcurrency  int              `json:"maxConcurrency"`  // zen 上游最大并发,防 worker 瞬时超限,默认 8
	Retries         int              `json:"retries"`         // 限流/网络错误重试次数,默认 3
	Failover        bool             `json:"failover"`        // zen 连续失败后故障转移到 cline 账号池,默认 true
	FailoverCount   int              `json:"failoverCount"`   // 触发故障转移的连续失败次数,默认 3
	FailoverMinutes int              `json:"failoverMinutes"` // 故障转移窗口(分钟),默认 5
	Compaction      zenCompactConfig `json:"compaction"`
}

func defaultZenConfig() *zenConfigData {
	return &zenConfigData{
		Enabled:         true,
		Key:             "public",
		BaseURL:         zenAPIBase,
		ProxyStrategy:   "round_robin",
		MaxConcurrency:  8,
		Retries:         3,
		Failover:        true,
		FailoverCount:   3,
		FailoverMinutes: 5,
		Compaction: zenCompactConfig{
			Auto:       true,
			Buffer:     20000,
			KeepTokens: 8000,
			MaxSummary: 4096,
		},
	}
}

var (
	zenConfig   = loadZenConfig()
	zenConfigMu sync.Mutex
)

// ============ 限流防御状态机 ============

var (
	zenSem       chan struct{} // 并发信号量
	zenFailCount int           // 连续失败计数
	zenFailUntil time.Time     // 故障转移截止时间
	zenStateMu   sync.Mutex
)

func init() {
	rebuildZenSem()
}

func rebuildZenSem() {
	cfg := getZenConfig()
	n := cfg.MaxConcurrency
	if n <= 0 {
		n = 8
	}
	zenStateMu.Lock()
	zenSem = make(chan struct{}, n)
	zenStateMu.Unlock()
}

func markZenSuccess() {
	zenStateMu.Lock()
	zenFailCount = 0
	zenFailUntil = time.Time{}
	zenStateMu.Unlock()
}

func markZenFail() {
	cfg := getZenConfig()
	thr := cfg.FailoverCount
	if thr <= 0 {
		thr = 3
	}
	window := cfg.FailoverMinutes
	if window <= 0 {
		window = 5
	}
	zenStateMu.Lock()
	zenFailCount++
	if zenFailCount >= thr {
		zenFailUntil = time.Now().Add(time.Duration(window) * time.Minute)
	}
	zenStateMu.Unlock()
}

// zenFailedNow zen 是否处于故障转移状态
func zenFailedNow() bool {
	zenStateMu.Lock()
	defer zenStateMu.Unlock()
	if zenFailUntil.IsZero() {
		return false
	}
	if time.Now().After(zenFailUntil) {
		zenFailCount = 0
		zenFailUntil = time.Time{}
		return false
	}
	return true
}

// isRateLimited 限流信号识别: 429/503 直接命中; 502/403 按错误体关键词
func isRateLimited(status int, body string) bool {
	if status == http.StatusTooManyRequests || status == http.StatusServiceUnavailable {
		return true
	}
	if status == http.StatusBadGateway || status == http.StatusForbidden {
		low := strings.ToLower(body)
		for _, kw := range []string{"resourceexhausted", "limit reached", "rate limit", "too many", "overloaded", "busy"} {
			if strings.Contains(low, kw) {
				return true
			}
		}
	}
	return false
}

func loadZenConfig() *zenConfigData {
	path := kit.ResolveDataPath(".zen-config.json")
	cfg := defaultZenConfig()
	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			log.Printf("zen config parse failed: %v", err)
		}
	}
	if cfg.Key == "" {
		cfg.Key = "public"
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = zenAPIBase
	}
	return cfg
}

func saveZenConfig() {
	zenConfigMu.Lock()
	defer zenConfigMu.Unlock()
	data, _ := json.MarshalIndent(zenConfig, "", "  ")
	if err := os.WriteFile(kit.ResolveDataPath(".zen-config.json"), data, 0600); err != nil {
		log.Printf("zen config save failed: %v", err)
	}
}

func getZenConfig() *zenConfigData {
	zenConfigMu.Lock()
	defer zenConfigMu.Unlock()
	return zenConfig
}

func setZenConfig(c *zenConfigData) {
	zenConfigMu.Lock()
	zenConfig = c
	zenConfigMu.Unlock()
	saveZenConfig()
	rebuildZenTransport()
	rebuildZenSem()
}

// ============ zen 上游调用 ============

// buildZenBody 构造 zen 请求体:只带 OpenAI 兼容字段,改写模型为 zen ID
func buildZenBody(params map[string]any, stream bool) map[string]any {
	body := map[string]any{}
	for _, key := range passThroughKeys {
		if val, ok := params[key]; ok {
			body[key] = val
		}
	}
	for _, key := range []string{"model", "messages", "max_tokens", "max_completion_tokens", "stream"} {
		if val, ok := params[key]; ok {
			body[key] = val
		}
	}
	if stream {
		body["stream"] = true
	}
	if model, ok := params["model"].(string); ok {
		if m, ok := resolveZenModel(model); ok {
			body["model"] = m.ID
		}
	}
	delete(body, "reasoning_effort")
	delete(body, "reasoningEffort")
	return body
}

// callZenAPI 调用 zen 上游,带限流防御: 并发信号量 + 指数退避重试 + 代理冷却 + 故障计数
// 返回 (响应, 命中限流次数, 错误)
func callZenAPI(params map[string]any, stream bool) (*http.Response, int, error) {
	cfg := getZenConfig()
	body := buildZenBody(params, stream)

	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("marshal zen body: %w", err)
	}

	endpoint := strings.TrimRight(cfg.BaseURL, "/") + "/chat/completions"

	zenStateMu.Lock()
	sem := zenSem
	sem <- struct{}{}
	zenStateMu.Unlock()
	defer func() { <-sem }()

	retries := cfg.Retries
	if retries <= 0 {
		retries = 3
	}
	delay := time.Second
	rateLimited := 0

	for attempt := 0; ; attempt++ {
		req, err := http.NewRequest("POST", endpoint, bytes.NewReader(bodyJSON))
		if err != nil {
			return nil, rateLimited, fmt.Errorf("create zen request: %w", err)
		}
		// 客户端身份轮换: 每次请求模拟全新 opencode 客户端,规避 session/UA 维度限流
		sess, user, ua := kit.FreshZenIdentity()
		req.Header.Set("Authorization", "Bearer "+cfg.Key)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", ua)
		req.Header.Set("x-opencode-session", sess)
		req.Header.Set("x-opencode-request", user)
		req.Header.Set("x-opencode-client", "cli")

		model, _ := params["model"].(string)
		if m, ok := resolveZenModel(model); ok {
			req.Header.Set("x-opencode-model", m.ID)
		}
		log.Printf("  zen upstream: model=%s stream=%v msgs=%d via=%s attempt=%d session=%s",
			body["model"], stream, getMsgCount(params), describeZenProxy(), attempt+1, kit.Truncate(sess, 24))

		resp, err := getZenHTTPClient().Do(req)
		if err != nil {
			// 网络错误:退避重试(不计入故障转移,瞬时可恢复)
			if attempt < retries {
				log.Printf("  zen network error (%v), retry %d/%d after %v", err, attempt+1, retries, delay)
				time.Sleep(kit.WithRetryJitter(delay))
				delay *= 2
				continue
			}
			return nil, rateLimited, fmt.Errorf("zen request: %w", err)
		}
		if resp.StatusCode == http.StatusOK {
			markZenSuccess()
			return resp, rateLimited, nil
		}

		bodyBytes := kit.ReadBody(resp)
		resp.Body.Close()
		reason := fmt.Sprintf("zen API %d: %s", resp.StatusCode, kit.Truncate(bodyBytes, 500))

		if isRateLimited(resp.StatusCode, bodyBytes) {
			rateLimited++
			// 冷却当前出口代理
			if idx := lastZenProxyIdx(); idx >= 0 {
				d := parseRetryAfter(resp.Header.Get("Retry-After"))
				if d <= 0 {
					d = 10 * time.Minute
				}
				cooldownZenProxy(idx, d)
				log.Printf("  zen rate limited (%d), proxy cooldown %v", resp.StatusCode, d)
			}
			if attempt < retries {
				wait := delay
				if retryAfter := parseRetryAfter(resp.Header.Get("Retry-After")); retryAfter > wait {
					wait = retryAfter
				}
				log.Printf("  zen rate limited (%d), retry %d/%d after %v", resp.StatusCode, attempt+1, retries, wait)
				time.Sleep(kit.WithRetryJitter(wait))
				delay *= 2
				continue
			}
			markZenFail()
			return nil, rateLimited, fmt.Errorf("%s", reason)
		}

		markZenFail()
		return nil, rateLimited, fmt.Errorf("%s", reason)
	}
}

func describeZenProxy() string {
	cfg := getZenConfig()
	proxies := cfg.Proxies
	if len(proxies) == 0 {
		return "direct"
	}
	idx := lastZenProxyIdx()
	if idx < 0 {
		idx = 0
	}
	idx %= len(proxies)
	return fmt.Sprintf("proxy[%d]=%s", idx+1, kit.Truncate(maskProxyURL(proxies[idx]), 60))
}

func zenModelList() []map[string]any {
	initZenModels()
	zenModelsMu.RLock()
	out := make([]map[string]any, 0, len(zenModels))
	for _, m := range zenModels {
		if !isZenFreeModel(m) {
			continue
		}
		cp := *m
		out = append(out, map[string]any{
			"id":      cp.ID,
			"context": cp.Context,
			"output":  cp.Output,
			"source":  cp.Source,
		})
	}
	zenModelsMu.RUnlock()
	return out
}

// syncZenModels 拉取 zen /v1/models,动态合并到模型表
func syncZenModels() (int, error) {
	initZenModels()
	cfg := getZenConfig()
	endpoint := strings.TrimRight(cfg.BaseURL, "/") + "/models"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.Key)
	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var payload struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, err
	}

	zenModelsMu.Lock()
	defer zenModelsMu.Unlock()
	added := 0
	for _, item := range payload.Data {
		id := item.ID
		if id == "" {
			continue
		}
		if _, ok := zenModels[id]; ok {
			continue
		}
		// 跳过与免费模型别名冲突的 ID(如付费的 deepseek-v4-flash),保证别名解析不被覆盖
		if _, conflict := zenAliases[id]; conflict {
			continue
		}
		// 新模型:默认按 200K 上下文接入,输出按 32K
		zenModels[id] = &ZenModel{
			ID:      id,
			Context: 200000,
			Output:  32768,
			Source:  "synced",
		}
		added++
	}
	return added, nil
}

// startZenModelsRefresher 定时同步 zen 模型列表(默认 10 分钟)
func startZenModelsRefresher() {
	go func() {
		if _, err := syncZenModels(); err != nil {
			log.Printf("zen model sync: failed (%v), using seed list", err)
		}
		ticker := time.NewTicker(10 * time.Minute)
		for range ticker.C {
			cfg := getZenConfig()
			if !cfg.Enabled {
				continue
			}
			if added, err := syncZenModels(); err != nil {
				log.Printf("zen model sync: failed (%v)", err)
			} else if added > 0 {
				log.Printf("zen model sync: %d new models from official feed", added)
			}
		}
	}()
}
