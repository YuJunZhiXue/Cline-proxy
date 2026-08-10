package app

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cline-go-proxy/internal/kit"
)

// RequestLog 单条代理请求记录（对话/API 调用历史）
type RequestLog struct {
	Time     time.Time `json:"time"`
	Client   string    `json:"client"`
	Method   string    `json:"method"`
	Path     string    `json:"path"`
	Model    string    `json:"model,omitempty"`
	Route    string    `json:"route"` // zen | cline | admin | other
	Status   int       `json:"status"`
	Duration int64     `json:"duration_ms"`
	Note     string    `json:"note,omitempty"`
}

const (
	maxReqLogs     = 500
	maxReqLogsFile = 10 << 20 // 10MB 上限，超出后清空落盘文件（内存仍保留最近 500 条）
)

var (
	reqLogsMu sync.Mutex
	reqLogs   []RequestLog
)

var reqLogsFile = kit.ResolveDataPath("requests.jsonl")

// AppendReqLog 记录一条请求日志：内存环形保留 + 异步追加落盘
func AppendReqLog(l RequestLog) {
	reqLogsMu.Lock()
	reqLogs = append(reqLogs, l)
	if len(reqLogs) > maxReqLogs {
		reqLogs = reqLogs[len(reqLogs)-maxReqLogs:]
	}
	reqLogsMu.Unlock()
	go func() {
		data, _ := json.Marshal(l)
		f, err := os.OpenFile(reqLogsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return
		}
		f.Write(append(data, '\n'))
		f.Close()
		if st, err := os.Stat(reqLogsFile); err == nil && st.Size() > maxReqLogsFile {
			os.WriteFile(reqLogsFile, nil, 0600)
		}
	}()
}

// LoadRequestLogs 返回最近的请求日志（内存优先，启动后从落盘文件补载）
func LoadRequestLogs() []RequestLog {
	reqLogsMu.Lock()
	defer reqLogsMu.Unlock()
	return reqLogs
}

// LoadRequestLogsFromFile 启动时从落盘文件读取尾部记录
func LoadRequestLogsFromFile() {
	raw, err := os.ReadFile(reqLogsFile)
	if err != nil {
		return
	}
	lines := splitLinesSafe(string(raw))
	for _, line := range lines {
		if line == "" {
			continue
		}
		var l RequestLog
		if json.Unmarshal([]byte(line), &l) == nil {
			reqLogs = append(reqLogs, l)
		}
	}
	if len(reqLogs) > maxReqLogs {
		reqLogs = reqLogs[len(reqLogs)-maxReqLogs:]
	}
}

func splitLinesSafe(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// ============ 请求日志中间件 ============

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// requestLogMiddleware 记录所有进入代理的请求（API 调用与对话历史）。
func requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}

		// 读取请求体提取模型，并放回，避免影响后续处理
		model := ""
		bodyBytes, _ := io.ReadAll(r.Body)
		if len(bodyBytes) > 0 {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
			var probe struct {
				Model string `json:"model"`
			}
			if json.Unmarshal(bodyBytes, &probe) == nil {
				model = probe.Model
			}
		}

		next.ServeHTTP(sw, r)

		if sw.status == 0 {
			sw.status = http.StatusOK
		}
		route := "other"
		switch {
		case strings.HasPrefix(r.URL.Path, "/admin"):
			route = "admin"
		case strings.HasPrefix(model, "zen/"):
			route = "zen"
		case model != "":
			route = "cline"
		case strings.Contains(r.URL.Path, "models") || strings.Contains(r.URL.Path, "health"):
			route = "meta"
		}
		client := r.RemoteAddr
		if host, _, err := net.SplitHostPort(client); err == nil {
			client = host
		}
		AppendReqLog(RequestLog{
			Time:     time.Now(),
			Client:   client,
			Method:   r.Method,
			Path:     r.URL.Path,
			Model:    model,
			Route:    route,
			Status:   sw.status,
			Duration: time.Since(start).Milliseconds(),
		})
	})
}
