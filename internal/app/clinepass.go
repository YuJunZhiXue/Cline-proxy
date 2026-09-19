package app

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"cline-go-proxy/internal/cline"
	"cline-go-proxy/internal/kit"
)

const (
	clinePassDefaultEndpoint = cline.ClineAPIBase + "/chat/completions"
	clinePassDefaultModel    = "cline-pass/deepseek-v4.1-flash"
)

// ClinePassAccount 是 API Key 身份，和 Cline OAuth Account 完全分离。
type ClinePassAccount struct {
	Name       string    `json:"name"`
	APIKey     string    `json:"apiKey"` // persisted in the 0600 data file; never returned by admin views
	Enabled    bool      `json:"enabled"`
	UsageCount int64     `json:"usageCount"`
	LastUsed   time.Time `json:"lastUsed,omitempty"`
}

type ClinePassModelPolicy struct {
	Pipeline  string   `json:"pipeline"` // auto / planner / direct
	Mode      string   `json:"mode"`     // auto / strict / preferred
	Providers []string `json:"providers,omitempty"`
	Exclude   []string `json:"exclude,omitempty"`
	Sort      string   `json:"sort"` // none / cost / ttft / tps
}

type ClinePassConfig struct {
	Enabled                     bool                            `json:"enabled"`
	BaseURL                     string                          `json:"baseURL,omitempty"`
	Accounts                    []ClinePassAccount              `json:"accounts"`
	AccountMode                 string                          `json:"accountMode"`
	CurrentIdx                  int                             `json:"currentIdx"`
	AllowClientProviderOverride bool                            `json:"allowClientProviderOverride"`
	PerModel                    map[string]ClinePassModelPolicy `json:"perModel"`
}

type ClinePassModelMetadata struct {
	Model               string            `json:"model"`
	Pipeline            string            `json:"pipeline"`
	DiscoveredProviders []string          `json:"discoveredProviders,omitempty"`
	ActualProvider      string            `json:"actualProvider"`
	ActualModel         string            `json:"actualModel"`
	ProviderStatus      map[string]string `json:"providerStatus,omitempty"`
	UpdatedAt           time.Time         `json:"updatedAt"`
}

type ClinePassAttempt struct {
	Provider string `json:"provider"`
	Status   int    `json:"status"`
	Kind     string `json:"kind,omitempty"`
}

type ClinePassRequestMeta struct {
	Upstream           string
	Model              string
	Pipeline           string
	RoutingMode        string
	RequestedProviders []string
	ActualProvider     string
	ActualModel        string
	Attempts           []ClinePassAttempt
}

type ClinePassResponseObservation struct {
	Pipeline       string
	ActualProvider string
	ActualModel    string
}

type clinePassUpstreamError struct {
	Status   int
	Kind     string
	Message  string
	Attempts []ClinePassAttempt
}

func (e *clinePassUpstreamError) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("clinepass upstream failed (%s)", e.Kind)
	}
	return e.Message
}

var (
	clinePassConfigPath = kit.ResolveDataPath(".clinepass-config.json")
	clinePassConfig     = loadClinePassConfig()
	clinePassConfigMu   sync.Mutex
	clinePassSaveMu     sync.Mutex

	clinePassMetadata   = make(map[string]*ClinePassModelMetadata)
	clinePassMetadataMu sync.RWMutex
)

func defaultClinePassConfig() *ClinePassConfig {
	return &ClinePassConfig{
		Enabled:                     false,
		BaseURL:                     clinePassDefaultEndpoint,
		Accounts:                    []ClinePassAccount{},
		AccountMode:                 "round_robin",
		AllowClientProviderOverride: false,
		PerModel:                    map[string]ClinePassModelPolicy{},
	}
}

func loadClinePassConfig() *ClinePassConfig {
	cfg := defaultClinePassConfig()
	data, err := osReadFile(clinePassConfigPath)
	if err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			log.Printf("clinepass config parse failed: %v", err)
		}
	}
	normalizeClinePassConfig(cfg)
	return cfg
}

// osReadFile is a small indirection that keeps config loading easy to isolate in tests.
var osReadFile = func(path string) ([]byte, error) { return os.ReadFile(path) }

func normalizeClinePassConfig(cfg *ClinePassConfig) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = clinePassDefaultEndpoint
	}
	if cfg.AccountMode != "single" && cfg.AccountMode != "round_robin" {
		cfg.AccountMode = "round_robin"
	}
	if cfg.Accounts == nil {
		cfg.Accounts = []ClinePassAccount{}
	}
	if cfg.PerModel == nil {
		cfg.PerModel = map[string]ClinePassModelPolicy{}
	}
	for model, policy := range cfg.PerModel {
		cfg.PerModel[model] = normalizeClinePassPolicy(policy)
	}
}

func normalizeClinePassPolicy(policy ClinePassModelPolicy) ClinePassModelPolicy {
	switch policy.Pipeline {
	case "planner", "direct":
	default:
		policy.Pipeline = "auto"
	}
	switch policy.Mode {
	case "strict", "preferred":
	default:
		policy.Mode = "auto"
	}
	switch policy.Sort {
	case "cost", "ttft", "tps":
	default:
		policy.Sort = "none"
	}
	policy.Providers = cleanProviderNames(policy.Providers)
	policy.Exclude = cleanProviderNames(policy.Exclude)
	return policy
}

func cleanProviderNames(in []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(in))
	for _, item := range in {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func cloneClinePassConfig(cfg *ClinePassConfig) *ClinePassConfig {
	cp := *cfg
	cp.Accounts = append([]ClinePassAccount(nil), cfg.Accounts...)
	cp.PerModel = make(map[string]ClinePassModelPolicy, len(cfg.PerModel))
	for model, policy := range cfg.PerModel {
		policy.Providers = append([]string(nil), policy.Providers...)
		policy.Exclude = append([]string(nil), policy.Exclude...)
		cp.PerModel[model] = policy
	}
	return &cp
}

func getClinePassConfig() *ClinePassConfig {
	clinePassConfigMu.Lock()
	defer clinePassConfigMu.Unlock()
	return cloneClinePassConfig(clinePassConfig)
}

func saveClinePassConfig() {
	clinePassConfigMu.Lock()
	cfg := cloneClinePassConfig(clinePassConfig)
	clinePassConfigMu.Unlock()

	clinePassSaveMu.Lock()
	defer clinePassSaveMu.Unlock()
	data, _ := json.MarshalIndent(cfg, "", "  ")
	if err := os.WriteFile(clinePassConfigPath, data, 0600); err != nil {
		log.Printf("clinepass config save failed: %v", err)
	}
}

func setClinePassConfig(cfg *ClinePassConfig) {
	normalizeClinePassConfig(cfg)
	clinePassConfigMu.Lock()
	clinePassConfig = cloneClinePassConfig(cfg)
	clinePassConfigMu.Unlock()
	saveClinePassConfig()
}

func clinePassHasModel(model string) bool {
	model = strings.TrimSpace(model)
	if model == "" {
		return false
	}
	cfg := getClinePassConfig()
	if !cfg.Enabled {
		return false
	}
	if _, ok := cfg.PerModel[model]; ok {
		return true
	}
	for pattern := range cfg.PerModel {
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(model, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	// cline-pass/ is a reserved namespace. It is only active when the independent
	// ClinePass provider is enabled, so ordinary Cline models remain unchanged.
	return strings.HasPrefix(strings.ToLower(model), "cline-pass/")
}

func clinePassPolicyForModel(model string) ClinePassModelPolicy {
	cfg := getClinePassConfig()
	if policy, ok := cfg.PerModel[model]; ok {
		return normalizeClinePassPolicy(policy)
	}
	var matched string
	for pattern := range cfg.PerModel {
		if strings.HasSuffix(pattern, "*") && strings.HasPrefix(model, strings.TrimSuffix(pattern, "*")) {
			if len(pattern) > len(matched) {
				matched = pattern
			}
		}
	}
	if matched != "" {
		return normalizeClinePassPolicy(cfg.PerModel[matched])
	}
	return normalizeClinePassPolicy(ClinePassModelPolicy{Pipeline: "auto", Mode: "auto", Sort: "none"})
}

func pickClinePassAccount() (ClinePassAccount, bool) {
	clinePassConfigMu.Lock()
	cfg := clinePassConfig
	if len(cfg.Accounts) == 0 {
		clinePassConfigMu.Unlock()
		return ClinePassAccount{}, false
	}
	idx := -1
	indexChanged := false
	if cfg.AccountMode == "single" {
		if cfg.CurrentIdx >= 0 && cfg.CurrentIdx < len(cfg.Accounts) {
			selected := cfg.Accounts[cfg.CurrentIdx]
			if selected.Enabled && strings.TrimSpace(selected.APIKey) != "" {
				idx = cfg.CurrentIdx
			}
		}
		if idx < 0 {
			for i := range cfg.Accounts {
				if cfg.Accounts[i].Enabled && strings.TrimSpace(cfg.Accounts[i].APIKey) != "" {
					idx = i
					if cfg.CurrentIdx != i {
						cfg.CurrentIdx = i
						indexChanged = true
					}
					break
				}
			}
		}
	} else {
		for i := 0; i < len(cfg.Accounts); i++ {
			candidate := (cfg.CurrentIdx + i) % len(cfg.Accounts)
			if cfg.Accounts[candidate].Enabled && strings.TrimSpace(cfg.Accounts[candidate].APIKey) != "" {
				idx = candidate
				cfg.CurrentIdx = (candidate + 1) % len(cfg.Accounts)
				break
			}
		}
	}
	if idx < 0 {
		clinePassConfigMu.Unlock()
		return ClinePassAccount{}, false
	}
	account := cfg.Accounts[idx]
	clinePassConfigMu.Unlock()
	// Persisting the index is intentionally outside the config lock and never
	// surrounds network I/O.
	if cfg.AccountMode != "single" || indexChanged {
		saveClinePassConfig()
	}
	return account, true
}

func recordClinePassUse(account ClinePassAccount) {
	clinePassConfigMu.Lock()
	for i := range clinePassConfig.Accounts {
		if clinePassConfig.Accounts[i].Name == account.Name && clinePassConfig.Accounts[i].APIKey == account.APIKey {
			clinePassConfig.Accounts[i].UsageCount++
			clinePassConfig.Accounts[i].LastUsed = time.Now()
			break
		}
	}
	clinePassConfigMu.Unlock()
	saveClinePassConfig()
}

func clinePassEndpoint(cfg *ClinePassConfig) string {
	base := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		return clinePassDefaultEndpoint
	}
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	return base + "/chat/completions"
}

func buildClinePassBody(params map[string]any, stream bool) map[string]any {
	body := map[string]any{}
	for _, key := range []string{"model", "messages", "max_tokens", "max_completion_tokens", "tools", "tool_choice", "parallel_tool_calls", "functions", "function_call", "temperature", "top_p", "top_k", "stop", "presence_penalty", "frequency_penalty", "response_format", "user", "n", "logit_bias", "seed", "logprobs", "top_logprobs", "stream_options", "metadata", "provider", "providerOptions"} {
		if val, ok := params[key]; ok {
			body[key] = val
		}
	}
	if _, ok := body["model"]; !ok {
		body["model"] = clinePassDefaultModel
	}
	if _, ok := body["max_tokens"]; !ok && body["max_completion_tokens"] == nil {
		body["max_tokens"] = defaultMaxTokens
	}
	if stream {
		body["stream"] = true
	} else if val, ok := params["stream"]; ok {
		body["stream"] = val
	}
	return body
}

func copyStringSlice(in []string) []string { return append([]string(nil), in...) }

// effectiveClinePassProviders is the single source of truth for the providers
// that a server policy may use. Every locally generated attempt must start from
// this list so an excluded provider cannot re-enter a fallback request.
func effectiveClinePassProviders(policy ClinePassModelPolicy) []string {
	policy = normalizeClinePassPolicy(policy)
	excluded := make(map[string]bool, len(policy.Exclude))
	for _, provider := range policy.Exclude {
		excluded[provider] = true
	}
	providers := make([]string, 0, len(policy.Providers))
	for _, provider := range policy.Providers {
		if !excluded[provider] {
			providers = append(providers, provider)
		}
	}
	return providers
}

func knownClinePassProviders(model string) []string {
	clinePassMetadataMu.RLock()
	defer clinePassMetadataMu.RUnlock()
	if m := clinePassMetadata[model]; m != nil {
		return copyStringSlice(m.DiscoveredProviders)
	}
	return nil
}

func applyClinePassProviderPolicy(body map[string]any, model string, policy ClinePassModelPolicy) map[string]any {
	// Client routing is never trusted by this helper. The caller may explicitly
	// choose to preserve it only when the global opt-in allows it.
	delete(body, "provider")
	delete(body, "providerOptions")
	policy = normalizeClinePassPolicy(policy)
	pipeline := policy.Pipeline
	if pipeline == "auto" {
		pipeline = "planner"
	}
	providers := effectiveClinePassProviders(policy)

	mode := policy.Mode
	if mode == "auto" {
		// AUTO does not pin a provider. When the server knows the candidate set,
		// Exclude is converted to a whitelist because gateway exclude support is
		// not consistent across upstream implementations.
		if len(policy.Exclude) == 0 {
			return body
		}
		if len(providers) == 0 {
			providers = knownClinePassProviders(model)
			knownPolicy := policy
			knownPolicy.Providers = providers
			providers = effectiveClinePassProviders(knownPolicy)
		}
		if len(providers) == 0 {
			return body
		}
		mode = "strict"
	}
	if len(providers) == 0 {
		return body
	}
	if mode == "preferred" {
		if pipeline == "direct" {
			body["provider"] = map[string]any{"order": providers, "only": providers}
			if policy.Sort != "none" {
				body["provider"].(map[string]any)["sort"] = clinePassDirectSort(policy.Sort)
			}
			return body
		}
		gateway := map[string]any{"order": providers, "only": providers}
		if policy.Sort != "none" {
			gateway["sort"] = policy.Sort
		}
		body["providerOptions"] = map[string]any{"gateway": gateway}
		return body
	}
	if pipeline == "direct" {
		provider := map[string]any{"only": providers}
		if policy.Sort != "none" {
			provider["sort"] = clinePassDirectSort(policy.Sort)
		}
		body["provider"] = provider
	} else {
		gateway := map[string]any{"only": providers}
		if policy.Sort != "none" {
			gateway["sort"] = policy.Sort
		}
		body["providerOptions"] = map[string]any{"gateway": gateway}
	}
	return body
}

func clinePassDirectSort(sortName string) string {
	switch sortName {
	case "cost":
		return "price"
	case "ttft":
		return "latency"
	case "tps":
		return "throughput"
	default:
		return ""
	}
}

func serverPolicyControlsRouting(policy ClinePassModelPolicy) bool {
	policy = normalizeClinePassPolicy(policy)
	return policy.Pipeline != "auto" || policy.Mode != "auto" || len(policy.Providers) > 0 || len(policy.Exclude) > 0 || policy.Sort != "none"
}

func resolveClinePassPipeline(model string, policy ClinePassModelPolicy) string {
	policy = normalizeClinePassPolicy(policy)
	if policy.Pipeline != "auto" {
		return policy.Pipeline
	}
	clinePassMetadataMu.RLock()
	if metadata := clinePassMetadata[model]; metadata != nil && (metadata.Pipeline == "planner" || metadata.Pipeline == "direct") {
		pipeline := metadata.Pipeline
		clinePassMetadataMu.RUnlock()
		return pipeline
	}
	clinePassMetadataMu.RUnlock()
	// planner is the safe first observation path; the response metadata can
	// subsequently classify the model as direct without changing client data.
	return "planner"
}

func requestedClinePassProviders(body map[string]any, policy ClinePassModelPolicy) []string {
	if len(policy.Providers) > 0 {
		return effectiveClinePassProviders(policy)
	}
	if provider, ok := body["provider"].(map[string]any); ok {
		if values, ok := provider["only"]; ok {
			if out := stringSliceValue(values); len(out) > 0 {
				return filterClinePassProviders(out, policy.Exclude)
			}
		}
		if values, ok := provider["order"]; ok {
			if out := stringSliceValue(values); len(out) > 0 {
				return filterClinePassProviders(out, policy.Exclude)
			}
		}
	}
	if options, ok := body["providerOptions"].(map[string]any); ok {
		if gateway, ok := options["gateway"].(map[string]any); ok {
			if values, ok := gateway["only"]; ok {
				if out := stringSliceValue(values); len(out) > 0 {
					return filterClinePassProviders(out, policy.Exclude)
				}
			}
			if values, ok := gateway["order"]; ok {
				if out := stringSliceValue(values); len(out) > 0 {
					return filterClinePassProviders(out, policy.Exclude)
				}
			}
		}
	}
	return nil
}

func filterClinePassProviders(providers, excluded []string) []string {
	if len(excluded) == 0 {
		return copyStringSlice(providers)
	}
	excludedSet := make(map[string]bool, len(excluded))
	for _, provider := range excluded {
		excludedSet[provider] = true
	}
	out := make([]string, 0, len(providers))
	for _, provider := range providers {
		if !excludedSet[provider] {
			out = append(out, provider)
		}
	}
	return out
}

func stringSliceValue(value any) []string {
	switch values := value.(type) {
	case []string:
		return copyStringSlice(values)
	case []any:
		return anyStrings(values)
	default:
		return nil
	}
}

func anyStrings(values []any) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if s, ok := value.(string); ok && strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

func makeClinePassAttemptPolicy(policy ClinePassModelPolicy, pipeline string, mode string, providers []string) ClinePassModelPolicy {
	return ClinePassModelPolicy{
		Pipeline:  pipeline,
		Mode:      mode,
		Providers: copyStringSlice(providers),
		Exclude:   copyStringSlice(policy.Exclude),
		Sort:      policy.Sort,
	}
}

func callClinePassAPI(params map[string]any, stream bool) (*http.Response, *ClinePassRequestMeta, error) {
	cfg := getClinePassConfig()
	if !cfg.Enabled {
		return nil, nil, &clinePassUpstreamError{Status: http.StatusServiceUnavailable, Kind: "upstream_error", Message: "clinepass upstream is disabled"}
	}
	account, ok := pickClinePassAccount()
	if !ok {
		return nil, nil, &clinePassUpstreamError{Status: http.StatusServiceUnavailable, Kind: "auth_error", Message: "no active clinepass API keys available"}
	}
	model, _ := params["model"].(string)
	serverPolicy := normalizeClinePassPolicy(clinePassPolicyForModel(model))
	pipeline := resolveClinePassPipeline(model, serverPolicy)
	baseBody := buildClinePassBody(params, stream)
	effectiveBody := baseBody
	controlsRouting := serverPolicyControlsRouting(serverPolicy)
	if !cfg.AllowClientProviderOverride || controlsRouting {
		effectiveBody = applyClinePassProviderPolicy(effectiveBody, model, ClinePassModelPolicy{
			Pipeline:  pipeline,
			Mode:      serverPolicy.Mode,
			Providers: serverPolicy.Providers,
			Exclude:   serverPolicy.Exclude,
			Sort:      serverPolicy.Sort,
		})
	}
	allowedProviders := effectiveClinePassProviders(serverPolicy)
	if len(allowedProviders) == 0 && len(serverPolicy.Exclude) > 0 {
		knownPolicy := serverPolicy
		knownPolicy.Providers = knownClinePassProviders(model)
		allowedProviders = effectiveClinePassProviders(knownPolicy)
	}
	requested := copyStringSlice(allowedProviders)
	if !controlsRouting {
		requested = requestedClinePassProviders(effectiveBody, ClinePassModelPolicy{})
	}
	meta := &ClinePassRequestMeta{
		Upstream:           "clinepass",
		Model:              model,
		Pipeline:           pipeline,
		RoutingMode:        normalizeClinePassPolicy(serverPolicy).Mode,
		RequestedProviders: copyStringSlice(requested),
		ActualProvider:     "unknown",
		ActualModel:        "unknown",
	}

	if (serverPolicy.Mode == "strict" || serverPolicy.Mode == "preferred" || len(serverPolicy.Exclude) > 0) && len(allowedProviders) == 0 {
		return nil, meta, &clinePassUpstreamError{
			Status:   http.StatusUnprocessableEntity,
			Kind:     "policy_error",
			Message:  "clinepass policy has no allowed providers",
			Attempts: meta.Attempts,
		}
	}

	// Preferred is intentionally controlled by the proxy. Each attempt is a
	// strict, single-provider request, so the upstream cannot add a provider
	// outside the configured order and cannot perform a second hidden fallback.
	policies := []ClinePassModelPolicy{}
	if serverPolicy.Mode == "preferred" && len(allowedProviders) > 0 {
		for _, provider := range allowedProviders {
			policies = append(policies, makeClinePassAttemptPolicy(serverPolicy, pipeline, "strict", []string{provider}))
		}
	} else {
		policies = append(policies, makeClinePassAttemptPolicy(serverPolicy, pipeline, serverPolicy.Mode, allowedProviders))
	}

	for i, policy := range policies {
		body := buildClinePassBody(params, stream)
		if i == 0 && cfg.AllowClientProviderOverride && !serverPolicyControlsRouting(serverPolicy) {
			body = effectiveBody
		} else {
			body = applyClinePassProviderPolicy(body, model, policy)
		}
		bodyJSON, err := json.Marshal(body)
		if err != nil {
			return nil, meta, &clinePassUpstreamError{Status: http.StatusInternalServerError, Kind: "upstream_error", Message: "marshal clinepass request failed", Attempts: meta.Attempts}
		}
		req, err := http.NewRequest(http.MethodPost, clinePassEndpoint(cfg), bytes.NewReader(bodyJSON))
		if err != nil {
			return nil, meta, &clinePassUpstreamError{Status: http.StatusInternalServerError, Kind: "upstream_error", Message: "create clinepass request failed", Attempts: meta.Attempts}
		}
		req.Header.Set("Authorization", "Bearer "+account.APIKey)
		req.Header.Set("Content-Type", "application/json")
		if stream {
			req.Header.Set("Accept", "text/event-stream")
		}

		attemptProvider := "auto"
		if policy.Mode != "auto" && len(policy.Providers) > 0 {
			attemptProvider = policy.Providers[0]
		}
		resp, err := kit.HTTPClient.Do(req)
		if err != nil {
			meta.Attempts = append(meta.Attempts, ClinePassAttempt{Provider: attemptProvider, Kind: "network_error"})
			if i+1 < len(policies) {
				continue
			}
			return nil, meta, &clinePassUpstreamError{Status: http.StatusBadGateway, Kind: "network_error", Message: "clinepass request failed: " + err.Error(), Attempts: meta.Attempts}
		}
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if !stream {
				bodyBytes, readErr := io.ReadAll(resp.Body)
				resp.Body.Close()
				if readErr != nil {
					meta.Attempts = append(meta.Attempts, ClinePassAttempt{Provider: attemptProvider, Status: resp.StatusCode, Kind: "network_error"})
					if i+1 < len(policies) {
						continue
					}
					return nil, meta, &clinePassUpstreamError{Status: http.StatusBadGateway, Kind: "network_error", Message: "clinepass response read failed", Attempts: meta.Attempts}
				}
				if clinePassResponseHasError(bodyBytes) {
					kind := classifyClinePassHTTPError(http.StatusBadGateway, string(bodyBytes))
					meta.Attempts = append(meta.Attempts, ClinePassAttempt{Provider: attemptProvider, Status: resp.StatusCode, Kind: kind})
					if i+1 < len(policies) {
						continue
					}
					return nil, meta, &clinePassUpstreamError{Status: http.StatusBadGateway, Kind: kind, Message: sanitizeClinePassError(string(bodyBytes), account.APIKey), Attempts: meta.Attempts}
				}
				resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			} else {
				prefix, errorBody, remainder, preflightErr := preflightClinePassStream(resp)
				if preflightErr != nil {
					_ = resp.Body.Close()
					meta.Attempts = append(meta.Attempts, ClinePassAttempt{Provider: attemptProvider, Status: resp.StatusCode, Kind: "network_error"})
					if i+1 < len(policies) {
						continue
					}
					return nil, meta, &clinePassUpstreamError{Status: http.StatusBadGateway, Kind: "network_error", Message: "clinepass response read failed", Attempts: meta.Attempts}
				}
				if errorBody != nil {
					_ = resp.Body.Close()
					kind := classifyClinePassHTTPError(http.StatusBadGateway, string(errorBody))
					meta.Attempts = append(meta.Attempts, ClinePassAttempt{Provider: attemptProvider, Status: resp.StatusCode, Kind: kind})
					if i+1 < len(policies) {
						continue
					}
					return nil, meta, &clinePassUpstreamError{
						Status:   http.StatusBadGateway,
						Kind:     kind,
						Message:  sanitizeClinePassError(string(errorBody), account.APIKey),
						Attempts: meta.Attempts,
					}
				}
				// The preflight has consumed bytes from the upstream. Put the
				// first event (and any buffered bytes) back in front of the
				// original stream so the client sees the response unchanged.
				resp.Body = io.NopCloser(io.MultiReader(bytes.NewReader(prefix), remainder))
			}
			meta.Attempts = append(meta.Attempts, ClinePassAttempt{Provider: attemptProvider, Status: resp.StatusCode, Kind: "ok"})
			recordClinePassUse(account)
			return resp, meta, nil
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		kind := classifyClinePassHTTPError(resp.StatusCode, string(bodyBytes))
		meta.Attempts = append(meta.Attempts, ClinePassAttempt{Provider: attemptProvider, Status: resp.StatusCode, Kind: kind})
		if i+1 < len(policies) {
			continue
		}
		return nil, meta, &clinePassUpstreamError{
			Status:   resp.StatusCode,
			Kind:     kind,
			Message:  sanitizeClinePassError(string(bodyBytes), account.APIKey),
			Attempts: meta.Attempts,
		}
	}
	return nil, meta, &clinePassUpstreamError{Status: http.StatusBadGateway, Kind: "upstream_error", Message: "clinepass upstream failed", Attempts: meta.Attempts}
}

// preflightClinePassStream checks a successful streaming response before the
// proxy has written anything to its client. A 2xx JSON error and an SSE error
// event are still upstream failures and may be retried by preferred mode.
// The returned remainder is the buffered reader itself, preserving bytes read
// ahead by bufio.Reader.
func preflightClinePassStream(resp *http.Response) (prefix, errorBody []byte, remainder io.Reader, err error) {
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "application/json") {
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, nil, nil, readErr
		}
		if clinePassResponseHasError(body) {
			return nil, body, nil, nil
		}
		return body, nil, strings.NewReader(""), nil
	}
	if contentType != "" && !strings.Contains(contentType, "text/event-stream") {
		return nil, nil, resp.Body, nil
	}

	prefix, data, reader, readErr := readClinePassSSEEvent(resp.Body)
	if readErr != nil {
		return prefix, nil, reader, readErr
	}
	if clinePassResponseHasError(data) {
		return prefix, data, reader, nil
	}
	return prefix, nil, reader, nil
}

func readClinePassSSEEvent(body io.Reader) ([]byte, []byte, io.Reader, error) {
	reader := bufio.NewReader(body)
	var raw bytes.Buffer
	var dataLines []string
	hasData := false
	for {
		line, readErr := reader.ReadBytes('\n')
		raw.Write(line)
		text := strings.TrimSuffix(strings.TrimSuffix(string(line), "\n"), "\r")
		if text == "" {
			if hasData {
				return append([]byte(nil), raw.Bytes()...), []byte(strings.Join(dataLines, "\n")), reader, nil
			}
		} else if strings.HasPrefix(text, "data:") {
			value := strings.TrimPrefix(text, "data:")
			if strings.HasPrefix(value, " ") {
				value = value[1:]
			}
			dataLines = append(dataLines, value)
			hasData = true
		}
		if readErr != nil {
			if readErr == io.EOF && hasData {
				return append([]byte(nil), raw.Bytes()...), []byte(strings.Join(dataLines, "\n")), reader, nil
			}
			return append([]byte(nil), raw.Bytes()...), nil, reader, readErr
		}
	}
}

func clinePassResponseHasError(body []byte) bool {
	var payload map[string]any
	if json.Unmarshal(body, &payload) != nil {
		return false
	}
	_, ok := payload["error"]
	return ok
}

func sanitizeClinePassError(message, apiKey string) string {
	if apiKey != "" {
		message = strings.ReplaceAll(message, apiKey, "[redacted]")
	}
	return kit.Truncate(strings.TrimSpace(message), 1000)
}

func classifyClinePassHTTPError(status int, body string) string {
	lower := strings.ToLower(body)
	if status == http.StatusUnauthorized || status == http.StatusForbidden || strings.Contains(lower, "invalid api key") || strings.Contains(lower, "authentication") {
		return "auth_error"
	}
	if status == http.StatusTooManyRequests || strings.Contains(lower, "rate limit") || strings.Contains(lower, "rate limited") || strings.Contains(lower, "temporarily rate limited") || strings.Contains(lower, "too many requests") {
		return "rate_limit"
	}
	if status == http.StatusBadRequest && (strings.Contains(lower, "provider") || strings.Contains(lower, "available_providers") || strings.Contains(lower, "not found")) {
		return "provider_invalid"
	}
	if status >= 500 {
		return "provider_unavailable"
	}
	return "upstream_error"
}

func observeClinePassPayload(model string, payload map[string]any) ClinePassResponseObservation {
	obs := ClinePassResponseObservation{}
	root := payload
	if nested, ok := payload["data"].(map[string]any); ok {
		root = nested
	}
	if actualModel, ok := root["model"].(string); ok && actualModel != "" {
		obs.ActualModel = actualModel
	} else if actualModel, ok := payload["model"].(string); ok && actualModel != "" {
		obs.ActualModel = actualModel
	}
	if metadata, ok := root["provider_metadata"].(map[string]any); ok {
		if gateway, ok := metadata["gateway"].(map[string]any); ok {
			if routing, ok := gateway["routing"].(map[string]any); ok {
				obs.Pipeline = "planner"
				obs.ActualProvider = firstString(routing, "finalProvider", "provider", "canonicalSlug")
				if routedModel := firstString(routing, "canonicalSlug", "model"); routedModel != "" {
					obs.ActualModel = routedModel
				}
			}
		}
	}
	if obs.ActualProvider == "" {
		if provider, ok := root["provider"]; ok {
			obs.Pipeline = "direct"
			obs.ActualProvider = providerName(provider)
		}
	}
	if obs.Pipeline == "" {
		if metadata, ok := payload["provider_metadata"].(map[string]any); ok {
			if gateway, ok := metadata["gateway"].(map[string]any); ok {
				if _, ok := gateway["routing"].(map[string]any); ok {
					obs.Pipeline = "planner"
				}
			}
		}
	}
	_ = model
	return obs
}

func firstString(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if s, ok := m[key].(string); ok && strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func providerName(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]any:
		return firstString(v, "name", "id", "slug", "provider")
	default:
		return ""
	}
}

func updateClinePassMetadata(model string, obs ClinePassResponseObservation, providers []string) {
	if model == "" {
		return
	}
	clinePassMetadataMu.Lock()
	metadata := clinePassMetadata[model]
	if metadata == nil {
		metadata = &ClinePassModelMetadata{Model: model, ProviderStatus: map[string]string{}}
		clinePassMetadata[model] = metadata
	}
	if obs.Pipeline == "planner" || obs.Pipeline == "direct" {
		metadata.Pipeline = obs.Pipeline
	}
	if obs.ActualProvider != "" {
		metadata.ActualProvider = obs.ActualProvider
	}
	if obs.ActualModel != "" {
		metadata.ActualModel = obs.ActualModel
	}
	if len(providers) > 0 {
		metadata.DiscoveredProviders = cleanProviderNames(append(metadata.DiscoveredProviders, providers...))
		sort.Strings(metadata.DiscoveredProviders)
	}
	metadata.UpdatedAt = time.Now()
	clinePassMetadataMu.Unlock()
}

func recordClinePassObservation(model string, meta *ClinePassRequestMeta, payload map[string]any) {
	obs := observeClinePassPayload(model, payload)
	if obs.Pipeline != "" {
		meta.Pipeline = obs.Pipeline
	}
	if obs.ActualProvider != "" {
		meta.ActualProvider = obs.ActualProvider
	}
	if obs.ActualModel != "" {
		meta.ActualModel = obs.ActualModel
	}
	if obs.ActualProvider == "" {
		obs.ActualProvider = meta.ActualProvider
	}
	if obs.ActualModel == "" {
		obs.ActualModel = meta.ActualModel
	}
	updateClinePassMetadata(model, obs, nil)
}

func writeClinePassError(w http.ResponseWriter, err error) {
	status := http.StatusBadGateway
	kind := "upstream_error"
	message := err.Error()
	var upstreamErr *clinePassUpstreamError
	if errors.As(err, &upstreamErr) {
		if upstreamErr.Status > 0 {
			status = upstreamErr.Status
		}
		kind = upstreamErr.Kind
		message = upstreamErr.Message
	}
	if status < 400 || status > 599 {
		status = http.StatusBadGateway
	}
	writeJSON(w, status, map[string]any{"error": map[string]string{"message": message, "type": kind}})
}

func handleClinePassChat(w http.ResponseWriter, r *http.Request, params map[string]any, isStream bool) {
	resp, meta, err := callClinePassAPI(params, isStream)
	if err != nil {
		if meta != nil {
			setRequestLogClinePass(r, meta)
		}
		writeClinePassError(w, err)
		return
	}
	defer resp.Body.Close()
	setRequestLogClinePass(r, meta)
	usageFn := func(map[string]any) {}
	if isStream {
		handleStreamResponseWithUsageMeta(w, resp, usageFn, func(payload map[string]any) {
			recordClinePassObservation(meta.Model, meta, payload)
			setRequestLogClinePass(r, meta)
		})
		return
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": map[string]string{"message": err.Error(), "type": "upstream_error"}})
		return
	}
	recordClinePassObservation(meta.Model, meta, raw)
	setRequestLogClinePass(r, meta)
	out := raw
	if data, ok := raw["data"].(map[string]any); ok {
		out = data
	}
	writeJSON(w, http.StatusOK, normalizeOpenAIResponse(out))
}

func handleClinePassResponses(w http.ResponseWriter, r *http.Request, params map[string]any, isStream bool) {
	chat := responsesToChat(params)
	resp, meta, err := callClinePassAPI(chat, isStream)
	if err != nil {
		if meta != nil {
			setRequestLogClinePass(r, meta)
		}
		writeClinePassError(w, err)
		return
	}
	defer resp.Body.Close()
	setRequestLogClinePass(r, meta)
	if isStream {
		chatStreamToResponsesWithMeta(w, resp, nil, func(payload map[string]any) {
			recordClinePassObservation(meta.Model, meta, payload)
			setRequestLogClinePass(r, meta)
		})
		return
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": map[string]string{"message": err.Error(), "type": "upstream_error"}})
		return
	}
	recordClinePassObservation(meta.Model, meta, raw)
	setRequestLogClinePass(r, meta)
	out := raw
	if data, ok := raw["data"].(map[string]any); ok {
		out = data
	}
	writeJSON(w, http.StatusOK, chatToResponses(normalizeOpenAIResponse(out)))
}

func handleClinePassAnthropic(w http.ResponseWriter, r *http.Request, req anthropicReq, openAIReq map[string]any, toolSchemas map[string]map[string]bool) {
	resp, meta, err := callClinePassAPI(openAIReq, req.Stream)
	if err != nil {
		if meta != nil {
			setRequestLogClinePass(r, meta)
		}
		writeClinePassError(w, err)
		return
	}
	defer resp.Body.Close()
	setRequestLogClinePass(r, meta)
	if req.Stream {
		handleAnthropicStreamWithMeta(w, resp, normalizeRequestModel(req.Model), toolSchemas, func(map[string]any) {}, func(payload map[string]any) {
			recordClinePassObservation(meta.Model, meta, payload)
			setRequestLogClinePass(r, meta)
		})
		return
	}
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]any{"error": map[string]string{"message": err.Error(), "type": "upstream_error"}})
		return
	}
	recordClinePassObservation(meta.Model, meta, raw)
	setRequestLogClinePass(r, meta)
	chatOut := raw
	if data, ok := raw["data"].(map[string]any); ok {
		chatOut = data
	}
	chatOut = normalizeOpenAIResponse(chatOut)
	writeJSON(w, http.StatusOK, openAIToAnthropic(chatOut))
}

var availableProvidersRE = regexp.MustCompile(`(?i)available[_ ]providers(?: are|:)?\s*[:：]?\s*([^\r\n]+)`)
var providerTokenRE = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9_.:/-]*`)

func parseAvailableClinePassProviders(body string) []string {
	var walk func(any) []string
	walk = func(value any) []string {
		var out []string
		switch v := value.(type) {
		case map[string]any:
			for key, child := range v {
				if strings.EqualFold(key, "available_providers") || strings.EqualFold(key, "availableProviders") {
					switch list := child.(type) {
					case []any:
						out = append(out, anyStrings(list)...)
					case map[string]any:
						for name := range list {
							out = append(out, name)
						}
					}
				}
				out = append(out, walk(child)...)
			}
		case []any:
			for _, child := range v {
				out = append(out, walk(child)...)
			}
		}
		return out
	}
	var decoded any
	if json.Unmarshal([]byte(body), &decoded) == nil {
		if providers := cleanProviderNames(walk(decoded)); len(providers) > 0 {
			return providers
		}
	}
	match := availableProvidersRE.FindStringSubmatch(body)
	if len(match) == 2 {
		values := providerTokenRE.FindAllString(match[1], -1)
		return cleanProviderNames(values)
	}
	return nil
}

func doClinePassHTTP(account ClinePassAccount, cfg *ClinePassConfig, body map[string]any) (*http.Response, string, error) {
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodPost, clinePassEndpoint(cfg), bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Authorization", "Bearer "+account.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := kit.HTTPClient.Do(req)
	if err != nil {
		return nil, "network_error", err
	}
	data, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()
	if readErr != nil {
		return nil, "network_error", readErr
	}
	return resp, string(data), nil
}

func probeClinePassProviders(model, requestedPipeline string) ([]string, string, error) {
	cfg := getClinePassConfig()
	account, ok := pickClinePassAccount()
	if !ok {
		return nil, "unknown", &clinePassUpstreamError{Status: http.StatusServiceUnavailable, Kind: "auth_error", Message: "no active clinepass API keys available"}
	}
	pipelines := []string{requestedPipeline}
	if requestedPipeline == "" || requestedPipeline == "auto" {
		pipelines = []string{"planner", "direct"}
	}
	var lastErr error
	for _, pipeline := range pipelines {
		body := buildClinePassBody(map[string]any{
			"model":      model,
			"messages":   []any{map[string]any{"role": "user", "content": "hi"}},
			"max_tokens": float64(16),
		}, false)
		if pipeline == "direct" {
			body["provider"] = map[string]any{"only": []string{"__probe__"}}
		} else {
			body["providerOptions"] = map[string]any{"gateway": map[string]any{"only": []string{"__probe__"}}}
		}
		resp, raw, err := doClinePassHTTP(account, cfg, body)
		if err != nil {
			lastErr = err
			continue
		}
		providers := parseAvailableClinePassProviders(raw)
		var decoded map[string]any
		if json.Unmarshal([]byte(raw), &decoded) == nil {
			obs := observeClinePassPayload(model, decoded)
			if obs.Pipeline != "" {
				pipeline = obs.Pipeline
			}
			updateClinePassMetadata(model, obs, providers)
		}
		if len(providers) > 0 {
			updateClinePassMetadata(model, ClinePassResponseObservation{Pipeline: pipeline}, providers)
			return providers, pipeline, nil
		}
		lastErr = fmt.Errorf("provider discovery returned no provider list (HTTP %d)", resp.StatusCode)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("provider discovery returned unknown")
	}
	return nil, "unknown", lastErr
}

func classifyClinePassProviderValidation(status int, body string) string {
	if status == http.StatusTooManyRequests || strings.Contains(strings.ToLower(body), "rate limit") || strings.Contains(strings.ToLower(body), "temporarily rate limited") {
		return "limited"
	}
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		return "auth_error"
	}
	if status == http.StatusBadRequest || status == http.StatusNotFound {
		return "bad"
	}
	if status >= 500 {
		return "provider_unavailable"
	}
	var payload map[string]any
	if json.Unmarshal([]byte(body), &payload) == nil {
		if _, ok := payload["error"]; ok {
			return "bad"
		}
	} else if strings.TrimSpace(body) != "" {
		return "unknown"
	}
	if status >= 400 {
		return "unknown"
	}
	return "ok"
}

func validateClinePassProviders(model string, providers []string) (map[string]string, error) {
	cfg := getClinePassConfig()
	account, ok := pickClinePassAccount()
	if !ok {
		return nil, fmt.Errorf("no active clinepass API keys available")
	}
	policy := clinePassPolicyForModel(model)
	pipeline := resolveClinePassPipeline(model, policy)
	if len(providers) == 0 {
		providers = knownClinePassProviders(model)
	}
	if len(providers) == 0 {
		return nil, fmt.Errorf("no discovered providers for model %q", model)
	}
	statuses := make(map[string]string, len(providers))
	for _, provider := range cleanProviderNames(providers) {
		body := buildClinePassBody(map[string]any{
			"model":      model,
			"messages":   []any{map[string]any{"role": "user", "content": "hi"}},
			"max_tokens": float64(16),
		}, false)
		body = applyClinePassProviderPolicy(body, model, makeClinePassAttemptPolicy(policy, pipeline, "strict", []string{provider}))
		resp, raw, err := doClinePassHTTP(account, cfg, body)
		if err != nil {
			statuses[provider] = "network_error"
			continue
		}
		statuses[provider] = classifyClinePassProviderValidation(resp.StatusCode, raw)
		var payload map[string]any
		if json.Unmarshal([]byte(raw), &payload) == nil {
			updateClinePassMetadata(model, observeClinePassPayload(model, payload), nil)
		}
	}
	clinePassMetadataMu.Lock()
	metadata := clinePassMetadata[model]
	if metadata == nil {
		metadata = &ClinePassModelMetadata{Model: model}
		clinePassMetadata[model] = metadata
	}
	if metadata.ProviderStatus == nil {
		metadata.ProviderStatus = map[string]string{}
	}
	for provider, status := range statuses {
		metadata.ProviderStatus[provider] = status
	}
	metadata.UpdatedAt = time.Now()
	clinePassMetadataMu.Unlock()
	return statuses, nil
}

func clinePassModels() []map[string]any {
	cfg := getClinePassConfig()
	clinePassMetadataMu.RLock()
	defer clinePassMetadataMu.RUnlock()
	models := make([]string, 0, len(cfg.PerModel)+len(clinePassMetadata))
	seen := map[string]bool{}
	for model := range cfg.PerModel {
		if strings.HasSuffix(model, "*") {
			continue
		}
		models = append(models, model)
		seen[model] = true
	}
	for model := range clinePassMetadata {
		if !seen[model] {
			models = append(models, model)
		}
	}
	sort.Strings(models)
	out := make([]map[string]any, 0, len(models))
	for _, model := range models {
		policy := cfg.PerModel[model]
		metadata := clinePassMetadata[model]
		item := map[string]any{
			"id":        model,
			"pipeline":  normalizeClinePassPolicy(policy).Pipeline,
			"mode":      normalizeClinePassPolicy(policy).Mode,
			"providers": policy.Providers,
			"exclude":   policy.Exclude,
			"sort":      policy.Sort,
		}
		if metadata != nil {
			item["pipeline"] = metadata.Pipeline
			item["actualProvider"] = metadata.ActualProvider
			item["actualModel"] = metadata.ActualModel
			item["providerStatus"] = metadata.ProviderStatus
			item["discoveredProviders"] = metadata.DiscoveredProviders
			item["updatedAt"] = metadata.UpdatedAt
		}
		out = append(out, item)
	}
	return out
}
