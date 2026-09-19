package app

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"

	"cline-go-proxy/internal/kit"
)

func installClinePassTestState(t *testing.T, cfg *ClinePassConfig) {
	t.Helper()
	oldCfg := getClinePassConfig()
	oldClient := kit.HTTPClient
	clinePassConfigMu.Lock()
	clinePassConfig = cloneClinePassConfig(cfg)
	clinePassConfigMu.Unlock()
	clinePassMetadataMu.Lock()
	oldMetadata := clinePassMetadata
	clinePassMetadata = make(map[string]*ClinePassModelMetadata)
	clinePassMetadataMu.Unlock()
	t.Cleanup(func() {
		clinePassConfigMu.Lock()
		clinePassConfig = oldCfg
		clinePassConfigMu.Unlock()
		clinePassMetadataMu.Lock()
		clinePassMetadata = oldMetadata
		clinePassMetadataMu.Unlock()
		kit.HTTPClient = oldClient
		_ = os.Remove(clinePassConfigPath)
	})
}

func testClinePassConfig(policy ClinePassModelPolicy) *ClinePassConfig {
	return &ClinePassConfig{
		Enabled:     true,
		BaseURL:     "http://127.0.0.1:1/chat/completions",
		AccountMode: "round_robin",
		Accounts: []ClinePassAccount{{
			Name:    "test",
			APIKey:  "sk-test-secret",
			Enabled: true,
		}},
		PerModel: map[string]ClinePassModelPolicy{
			"cline-pass/test": policy,
		},
	}
}

func decodeRequestBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return body
}

func TestApplyClinePassProviderPolicy(t *testing.T) {
	tests := []struct {
		name   string
		policy ClinePassModelPolicy
		assert func(t *testing.T, body map[string]any)
	}{
		{
			name:   "planner strict",
			policy: ClinePassModelPolicy{Pipeline: "planner", Mode: "strict", Providers: []string{"deepseek"}},
			assert: func(t *testing.T, body map[string]any) {
				options := body["providerOptions"].(map[string]any)
				gateway := options["gateway"].(map[string]any)
				if got := gateway["only"].([]string); len(got) != 1 || got[0] != "deepseek" {
					t.Fatalf("unexpected planner only: %#v", got)
				}
				if _, ok := body["provider"]; ok {
					t.Fatal("direct provider leaked into planner body")
				}
			},
		},
		{
			name:   "direct strict",
			policy: ClinePassModelPolicy{Pipeline: "direct", Mode: "strict", Providers: []string{"deepseek"}},
			assert: func(t *testing.T, body map[string]any) {
				provider := body["provider"].(map[string]any)
				if got := provider["only"].([]string); len(got) != 1 || got[0] != "deepseek" {
					t.Fatalf("unexpected direct only: %#v", got)
				}
				if _, ok := body["providerOptions"]; ok {
					t.Fatal("planner options leaked into direct body")
				}
			},
		},
		{
			name:   "planner preferred sorted",
			policy: ClinePassModelPolicy{Pipeline: "planner", Mode: "preferred", Providers: []string{"deepseek", "novita"}, Sort: "cost"},
			assert: func(t *testing.T, body map[string]any) {
				gateway := body["providerOptions"].(map[string]any)["gateway"].(map[string]any)
				got := gateway["order"].([]string)
				only := gateway["only"].([]string)
				if len(got) != 2 || got[0] != "deepseek" || got[1] != "novita" || len(only) != 2 || only[0] != "deepseek" || only[1] != "novita" || gateway["sort"] != "cost" {
					t.Fatalf("unexpected planner preferred: %#v", gateway)
				}
			},
		},
		{
			name:   "direct preferred",
			policy: ClinePassModelPolicy{Pipeline: "direct", Mode: "preferred", Providers: []string{"deepseek", "novita"}},
			assert: func(t *testing.T, body map[string]any) {
				provider := body["provider"].(map[string]any)
				got := provider["order"].([]string)
				only := provider["only"].([]string)
				if len(got) != 2 || got[0] != "deepseek" || got[1] != "novita" || len(only) != 2 || only[0] != "deepseek" || only[1] != "novita" {
					t.Fatalf("unexpected direct preferred: %#v", provider)
				}
			},
		},
		{
			name:   "auto does not pin",
			policy: ClinePassModelPolicy{Pipeline: "planner", Mode: "auto", Providers: []string{"deepseek"}},
			assert: func(t *testing.T, body map[string]any) {
				if _, ok := body["provider"]; ok {
					t.Fatal("auto unexpectedly pinned direct provider")
				}
				if _, ok := body["providerOptions"]; ok {
					t.Fatal("auto unexpectedly pinned gateway provider")
				}
			},
		},
		{
			name:   "exclude filters known candidates",
			policy: ClinePassModelPolicy{Pipeline: "direct", Mode: "auto", Providers: []string{"deepseek", "novita"}, Exclude: []string{"novita"}},
			assert: func(t *testing.T, body map[string]any) {
				provider := body["provider"].(map[string]any)
				got := provider["only"].([]string)
				if len(got) != 1 || got[0] != "deepseek" {
					t.Fatalf("excluded provider was not removed: %#v", provider)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := map[string]any{
				"model":           "cline-pass/test",
				"provider":        map[string]any{"only": []string{"novita"}},
				"providerOptions": map[string]any{"gateway": map[string]any{"only": []string{"novita"}}},
			}
			tt.assert(t, applyClinePassProviderPolicy(body, "cline-pass/test", tt.policy))
		})
	}
}

func TestClinePassClientRoutingCannotOverrideServerPolicy(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "direct", Mode: "strict", Providers: []string{"deepseek"}})
	installClinePassTestState(t, cfg)
	body := buildClinePassBody(map[string]any{
		"model":           "cline-pass/test",
		"messages":        []any{},
		"provider":        map[string]any{"only": []string{"novita"}},
		"providerOptions": map[string]any{"gateway": map[string]any{"only": []string{"novita"}}},
	}, false)
	final := applyClinePassProviderPolicy(body, "cline-pass/test", cfg.PerModel["cline-pass/test"])
	provider := final["provider"].(map[string]any)
	if got := provider["only"].([]string); len(got) != 1 || got[0] != "deepseek" {
		t.Fatalf("client routing bypassed server policy: %#v", provider)
	}
}

func TestClinePassPipelineAndMetadataDetection(t *testing.T) {
	planner := map[string]any{"model": "cline-pass/test", "provider_metadata": map[string]any{"gateway": map[string]any{"routing": map[string]any{"canonicalSlug": "deepseek-v4.1-flash", "finalProvider": "deepseek"}}}}
	obs := observeClinePassPayload("cline-pass/test", planner)
	if obs.Pipeline != "planner" || obs.ActualProvider != "deepseek" || obs.ActualModel != "deepseek-v4.1-flash" {
		t.Fatalf("planner observation mismatch: %#v", obs)
	}
	direct := map[string]any{"model": "deepseek-v4.1-flash", "provider": "deepseek"}
	obs = observeClinePassPayload("cline-pass/test", direct)
	if obs.Pipeline != "direct" || obs.ActualProvider != "deepseek" || obs.ActualModel != "deepseek-v4.1-flash" {
		t.Fatalf("direct observation mismatch: %#v", obs)
	}
	unknown := observeClinePassPayload("cline-pass/test", map[string]any{"choices": []any{}})
	if unknown.Pipeline != "" || unknown.ActualProvider != "" || unknown.ActualModel != "" {
		t.Fatalf("unknown observation should be empty: %#v", unknown)
	}
}

func TestClinePassPreferredFallbackAfter429(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "planner", Mode: "preferred", Providers: []string{"deepseek", "novita"}})
	installClinePassTestState(t, cfg)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		body := decodeRequestBody(t, r)
		if call == 1 {
			options, ok := body["providerOptions"].(map[string]any)
			if !ok {
				t.Errorf("first request missing providerOptions: %#v", body)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			gateway, ok := options["gateway"].(map[string]any)
			if !ok {
				t.Errorf("first request missing gateway options: %#v", body)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			only, ok := gateway["only"].([]any)
			if !ok {
				t.Errorf("first request missing only: %#v", gateway)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if len(only) != 1 || only[0] != "deepseek" {
				t.Errorf("first request did not pin deepseek: %#v", gateway)
			}
			if _, ok := gateway["order"]; ok {
				t.Errorf("proxy-controlled preferred request unexpectedly used gateway order: %#v", gateway)
			}
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"temporarily rate limited"}`))
			return
		}
		providerOptions := body["providerOptions"].(map[string]any)
		provider := providerOptions["gateway"].(map[string]any)
		only := provider["only"].([]any)
		if len(only) != 1 || only[0] != "novita" {
			t.Errorf("fallback request did not pin novita: %#v", provider)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"deepseek-v4.1-flash","provider":"novita","choices":[]}`))
	}))
	defer server.Close()
	cfg.BaseURL = server.URL
	clinePassConfigMu.Lock()
	clinePassConfig.BaseURL = cfg.BaseURL
	clinePassConfigMu.Unlock()
	kit.HTTPClient = server.Client()
	resp, meta, err := callClinePassAPI(map[string]any{"model": "cline-pass/test", "messages": []any{}}, false)
	if err != nil {
		t.Fatalf("fallback returned error: %v", err)
	}
	defer resp.Body.Close()
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected two attempts, got %d", got)
	}
	if len(meta.Attempts) != 2 || meta.Attempts[0].Kind != "rate_limit" || meta.Attempts[1].Provider != "novita" {
		t.Fatalf("unexpected attempts: %#v", meta.Attempts)
	}
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestClinePassPreferredFallbackAfterNetworkError(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "direct", Mode: "preferred", Providers: []string{"deepseek", "novita"}})
	installClinePassTestState(t, cfg)
	var calls int32
	kit.HTTPClient = &http.Client{Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		call := atomic.AddInt32(&calls, 1)
		body := decodeRequestBody(t, r)
		provider := body["provider"].(map[string]any)
		only := provider["only"].([]any)
		expected := "deepseek"
		if call == 2 {
			expected = "novita"
		}
		if len(only) != 1 || only[0] != expected {
			t.Errorf("direct preferred attempt %d used %#v, want only=%s", call, provider, expected)
		}
		if _, ok := provider["order"]; ok {
			t.Errorf("direct preferred attempt %d unexpectedly used order: %#v", call, provider)
		}
		if call == 1 {
			return nil, errors.New("simulated dial failure")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"model":"deepseek-v4.1-flash","provider":"novita","choices":[]}`)),
		}, nil
	})}
	resp, meta, err := callClinePassAPI(map[string]any{"model": "cline-pass/test", "messages": []any{}}, false)
	if err != nil {
		t.Fatalf("network fallback returned error: %v", err)
	}
	defer resp.Body.Close()
	if calls != 2 || len(meta.Attempts) != 2 || meta.Attempts[0].Kind != "network_error" || meta.Attempts[1].Provider != "novita" {
		t.Fatalf("unexpected network fallback attempts: calls=%d meta=%#v", calls, meta.Attempts)
	}
}

func TestClinePassPreferredExcludeNeverRetriesExcludedProvider(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{
		Pipeline:  "planner",
		Mode:      "preferred",
		Providers: []string{"deepseek", "novita", "fireworks"},
		Exclude:   []string{"novita"},
	})
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := atomic.AddInt32(&calls, 1)
		body := decodeRequestBody(t, r)
		raw, _ := json.Marshal(body)
		if strings.Contains(string(raw), "novita") {
			t.Errorf("excluded provider appeared in request %d: %s", call, raw)
		}
		gateway := body["providerOptions"].(map[string]any)["gateway"].(map[string]any)
		only := gateway["only"].([]any)
		expected := "deepseek"
		if call == 2 {
			expected = "fireworks"
		}
		if len(only) != 1 || only[0] != expected {
			t.Errorf("request %d used %v, want only=%s", call, only, expected)
		}
		if call == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"temporarily rate limited"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"deepseek-v4.1-flash","provider":"fireworks","choices":[]} `))
	}))
	defer server.Close()
	cfg.BaseURL = server.URL
	installClinePassTestState(t, cfg)
	kit.HTTPClient = server.Client()

	resp, meta, err := callClinePassAPI(map[string]any{"model": "cline-pass/test", "messages": []any{}}, false)
	if err != nil {
		t.Fatalf("preferred exclude fallback returned error: %v", err)
	}
	_ = resp.Body.Close()
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected two attempts, got %d", got)
	}
	if len(meta.RequestedProviders) != 2 || meta.RequestedProviders[0] != "deepseek" || meta.RequestedProviders[1] != "fireworks" {
		t.Fatalf("excluded provider remained in requested providers: %#v", meta.RequestedProviders)
	}
	if len(meta.Attempts) != 2 || meta.Attempts[0].Provider != "deepseek" || meta.Attempts[1].Provider != "fireworks" {
		t.Fatalf("unexpected attempts: %#v", meta.Attempts)
	}
}

func TestClinePassDeepSeekStrictNeverFallsBack(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{
		Pipeline:  "planner",
		Mode:      "strict",
		Providers: []string{"deepseek"},
		Exclude:   []string{"novita", "fireworks"},
	})
	cfg.PerModel = map[string]ClinePassModelPolicy{
		"cline-pass/deepseek-v4.1-flash": cfg.PerModel["cline-pass/test"],
	}
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		body := decodeRequestBody(t, r)
		options := body["providerOptions"].(map[string]any)
		gateway := options["gateway"].(map[string]any)
		only := gateway["only"].([]any)
		if len(only) != 1 || only[0] != "deepseek" {
			t.Errorf("unexpected strict provider policy: %#v", gateway)
		}
		if _, ok := gateway["order"]; ok {
			t.Errorf("strict request unexpectedly contained order: %#v", gateway)
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()
	cfg.BaseURL = server.URL
	installClinePassTestState(t, cfg)
	kit.HTTPClient = server.Client()

	_, _, err := callClinePassAPI(map[string]any{"model": "cline-pass/deepseek-v4.1-flash", "messages": []any{}}, false)
	if err == nil {
		t.Fatal("strict request unexpectedly succeeded")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("strict request fell back after %d upstream calls", got)
	}
}

func TestClinePassClientRoutingCannotOverrideServerPolicyAtCallLevel(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "planner", Mode: "strict", Providers: []string{"deepseek"}})
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		body := decodeRequestBody(t, r)
		gateway := body["providerOptions"].(map[string]any)["gateway"].(map[string]any)
		only := gateway["only"].([]any)
		if len(only) != 1 || only[0] != "deepseek" {
			t.Errorf("client provider override reached upstream: %#v", gateway)
		}
		raw, _ := json.Marshal(body)
		if strings.Contains(string(raw), "novita") {
			t.Errorf("client provider leaked to upstream: %s", raw)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"deepseek-v4.1-flash","provider":"deepseek","choices":[]} `))
	}))
	defer server.Close()
	cfg.BaseURL = server.URL
	installClinePassTestState(t, cfg)
	kit.HTTPClient = server.Client()

	resp, _, err := callClinePassAPI(map[string]any{
		"model":           "cline-pass/test",
		"providerOptions": map[string]any{"gateway": map[string]any{"only": []any{"novita"}}},
	}, false)
	if err != nil {
		t.Fatalf("call-level policy test failed: %v", err)
	}
	_ = resp.Body.Close()
	if calls != 1 {
		t.Fatalf("expected one upstream request, got %d", calls)
	}
}

func TestClinePassSingleUsesConfiguredCurrentAccount(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "direct", Mode: "strict", Providers: []string{"deepseek"}})
	cfg.AccountMode = "single"
	cfg.CurrentIdx = 1
	cfg.Accounts = []ClinePassAccount{
		{Name: "first", APIKey: "sk-first", Enabled: true},
		{Name: "selected", APIKey: "sk-selected", Enabled: true},
		{Name: "third", APIKey: "sk-third", Enabled: true},
	}
	installClinePassTestState(t, cfg)
	account, ok := pickClinePassAccount()
	if !ok || account.Name != "selected" {
		t.Fatalf("single mode selected %#v, want selected account", account)
	}
}

func TestClinePassDirectSortMapping(t *testing.T) {
	for input, expected := range map[string]string{"cost": "price", "ttft": "latency", "tps": "throughput"} {
		body := applyClinePassProviderPolicy(map[string]any{}, "cline-pass/test", ClinePassModelPolicy{
			Pipeline:  "direct",
			Mode:      "preferred",
			Providers: []string{"deepseek"},
			Sort:      input,
		})
		provider := body["provider"].(map[string]any)
		if got := provider["sort"]; got != expected {
			t.Fatalf("direct sort %q mapped to %v, want %s", input, got, expected)
		}
	}
}

func TestClinePassStreamingJSONErrorPreflightFallsBack(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "planner", Mode: "preferred", Providers: []string{"deepseek", "novita"}})
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"error":"provider unavailable"}`))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()
	cfg.BaseURL = server.URL
	installClinePassTestState(t, cfg)
	kit.HTTPClient = server.Client()

	resp, meta, err := callClinePassAPI(map[string]any{"model": "cline-pass/test", "stream": true}, true)
	if err != nil {
		t.Fatalf("JSON streaming preflight did not fallback: %v", err)
	}
	defer resp.Body.Close()
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected JSON error fallback, got %d calls", got)
	}
	if len(meta.Attempts) != 2 || meta.Attempts[0].Provider != "deepseek" || meta.Attempts[1].Provider != "novita" {
		t.Fatalf("unexpected JSON preflight attempts: %#v", meta.Attempts)
	}
}

func TestClinePassStreamingSSEErrorPreflightFallsBack(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "planner", Mode: "preferred", Providers: []string{"deepseek", "novita"}})
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = w.Write([]byte("data: {\"error\":\"rate limited\"}\n\n"))
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n"))
	}))
	defer server.Close()
	cfg.BaseURL = server.URL
	installClinePassTestState(t, cfg)
	kit.HTTPClient = server.Client()

	resp, meta, err := callClinePassAPI(map[string]any{"model": "cline-pass/test", "stream": true}, true)
	if err != nil {
		t.Fatalf("SSE error preflight did not fallback: %v", err)
	}
	_ = resp.Body.Close()
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("expected SSE error fallback, got %d calls", got)
	}
	if len(meta.Attempts) != 2 || meta.Attempts[0].Kind != "rate_limit" || meta.Attempts[1].Kind != "ok" {
		t.Fatalf("unexpected SSE preflight attempts: %#v", meta.Attempts)
	}
}

func TestClinePassStreamingNormalFirstEventDisablesFallback(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "planner", Mode: "preferred", Providers: []string{"deepseek", "novita"}})
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\ndata: {\"error\":\"late failure\"}\n\n"))
	}))
	defer server.Close()
	cfg.BaseURL = server.URL
	installClinePassTestState(t, cfg)
	kit.HTTPClient = server.Client()

	resp, _, err := callClinePassAPI(map[string]any{"model": "cline-pass/test", "stream": true}, true)
	if err != nil {
		t.Fatalf("normal first SSE event was treated as failure: %v", err)
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(body), "late failure") || atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("stream retried after first event: calls=%d body=%s", calls, body)
	}
}

func TestClinePassStreamingPreflightPreservesFirstEvent(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "planner", Mode: "strict", Providers: []string{"deepseek"}})
	expected := "data: {\"choices\":[{\"delta\":{\"content\":\"first\"}}]}\n\ndata: {\"choices\":[{\"delta\":{\"content\":\"second\"}}]}\n\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(expected))
	}))
	defer server.Close()
	cfg.BaseURL = server.URL
	installClinePassTestState(t, cfg)
	kit.HTTPClient = server.Client()

	resp, _, err := callClinePassAPI(map[string]any{"model": "cline-pass/test", "stream": true}, true)
	if err != nil {
		t.Fatalf("SSE preflight failed: %v", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != expected {
		t.Fatalf("preflight dropped or changed first event: got %q want %q", body, expected)
	}
}

func TestClinePassProtocolEntrypoints(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "direct", Mode: "strict", Providers: []string{"deepseek"}})
	installClinePassTestState(t, cfg)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := decodeRequestBody(t, r)
		provider := body["provider"].(map[string]any)
		if provider["only"].([]any)[0] != "deepseek" {
			t.Errorf("unexpected provider policy: %#v", provider)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"model":"deepseek-v4.1-flash","provider":"deepseek","choices":[{"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer server.Close()
	clinePassConfigMu.Lock()
	clinePassConfig.BaseURL = server.URL
	clinePassConfigMu.Unlock()
	kit.HTTPClient = server.Client()

	chatReq := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(`{"model":"cline-pass/test","messages":[{"role":"user","content":"hi"}]}`))
	chatW := httptest.NewRecorder()
	handleClinePassChat(chatW, chatReq, map[string]any{"model": "cline-pass/test", "messages": []any{}}, false)
	if chatW.Code != http.StatusOK || !strings.Contains(chatW.Body.String(), `"content":"ok"`) {
		t.Fatalf("chat entrypoint failed: %d %s", chatW.Code, chatW.Body.String())
	}

	responsesReq := httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(`{"model":"cline-pass/test","input":"hi"}`))
	responsesW := httptest.NewRecorder()
	handleResponses(responsesW, responsesReq)
	if responsesW.Code != http.StatusOK || !strings.Contains(responsesW.Body.String(), `"output_text":"ok"`) {
		t.Fatalf("responses entrypoint failed: %d %s", responsesW.Code, responsesW.Body.String())
	}

	anthropicReq := httptest.NewRequest(http.MethodPost, "/v1/messages", strings.NewReader(`{"model":"cline-pass/test","max_tokens":16,"messages":[{"role":"user","content":"hi"}]}`))
	anthropicW := httptest.NewRecorder()
	handleAnthropicMessages(anthropicW, anthropicReq)
	if anthropicW.Code != http.StatusOK || !strings.Contains(anthropicW.Body.String(), `"text":"ok"`) {
		t.Fatalf("anthropic entrypoint failed: %d %s", anthropicW.Code, anthropicW.Body.String())
	}
}

type flushRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (w *flushRecorder) Flush() {
	w.flushes++
	w.ResponseRecorder.Flush()
}

func TestClinePassSSEUsesFlusherAndDoesNotRetryAfterBodyStarts(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "planner", Mode: "preferred", Providers: []string{"deepseek", "novita"}})
	installClinePassTestState(t, cfg)
	var calls int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"model\":\"deepseek-v4.1-flash\",\"provider_metadata\":{\"gateway\":{\"routing\":{\"finalProvider\":\"deepseek\"}}},\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n"))
	}))
	defer server.Close()
	clinePassConfigMu.Lock()
	clinePassConfig.BaseURL = server.URL
	clinePassConfigMu.Unlock()
	kit.HTTPClient = server.Client()
	resp, meta, err := callClinePassAPI(map[string]any{"model": "cline-pass/test", "messages": []any{}, "stream": true}, true)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	w := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	handleStreamResponseWithUsageMeta(w, resp, nil, func(payload map[string]any) {
		recordClinePassObservation(meta.Model, meta, payload)
	})
	if w.flushes == 0 {
		t.Fatal("streaming response did not flush")
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Fatalf("stream started and unexpectedly retried: %d requests", got)
	}
	if meta.ActualProvider != "deepseek" {
		t.Fatalf("actual provider not observed from SSE: %#v", meta)
	}
}

func TestClinePassProviderDiscoveryParsingAndKeyMasking(t *testing.T) {
	providers := parseAvailableClinePassProviders(`{"error":{"message":"Available providers are: deepseek, novita, fireworks"}}`)
	if len(providers) != 3 || providers[0] != "deepseek" || providers[2] != "fireworks" {
		t.Fatalf("unexpected discovered providers: %#v", providers)
	}
	if got := maskClinePassKey("sk-abcd123456xyz"); got != "sk-abcd****xyz" {
		t.Fatalf("unexpected masked key: %s", got)
	}
	if strings.Contains(maskClinePassKey("sk-abcd123456xyz"), "123456") {
		t.Fatal("masked key leaked secret middle")
	}
}

func TestClinePassAdminDoesNotExposeAPIKey(t *testing.T) {
	cfg := testClinePassConfig(ClinePassModelPolicy{Pipeline: "planner", Mode: "strict", Providers: []string{"deepseek"}})
	installClinePassTestState(t, cfg)
	w := httptest.NewRecorder()
	handleClinePassAccounts(w, httptest.NewRequest(http.MethodGet, "/admin/api/clinepass/accounts", nil))
	if strings.Contains(w.Body.String(), "sk-test-secret") {
		t.Fatalf("admin account API leaked full key: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "sk-test****") {
		t.Fatalf("admin account API did not return a masked key: %s", w.Body.String())
	}

	w = httptest.NewRecorder()
	handleClinePassConfig(w, httptest.NewRequest(http.MethodGet, "/admin/api/clinepass/config", nil))
	if strings.Contains(w.Body.String(), "sk-test-secret") || strings.Contains(w.Body.String(), "apiKey") {
		t.Fatalf("admin config API exposed key material: %s", w.Body.String())
	}

	w = httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPost, "/admin/api/clinepass/config/update", strings.NewReader(`{"currentIdx":0}`))
	handleClinePassConfigUpdate(w, updateReq)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"currentIdx":0`) {
		t.Fatalf("admin current account update failed: %d %s", w.Code, w.Body.String())
	}
}
