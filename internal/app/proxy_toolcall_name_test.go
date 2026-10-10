package app

import (
	"reflect"
	"testing"
)

func tcOf(id, name, args string) map[string]any {
	return map[string]any{
		"id":       id,
		"type":     "function",
		"function": map[string]any{"name": name, "arguments": args},
	}
}

// TestCleanMessagesDropsEmptyNameWithResult 复现线上故障形态：assistant 带一个
// function.name 为空的 tool_call，且带配对的 role=tool 结果（旧实现里它靠 id 配对
// 得以原样出站，被严格校验的上游以
// "tool_calls[0].function.name must be a non-empty string" 判 400，整条会话后续每轮都挂）。
// 修复后必须：脏调用被剔除、其配对 tool 结果一并删（否则成孤儿、上游判配对断裂），
// 而 assistant 消息本身保留、其余消息顺序不变。
func TestCleanMessagesDropsEmptyNameWithResult(t *testing.T) {
	in := []any{
		map[string]any{"role": "user", "content": "hi"},
		map[string]any{"role": "assistant", "content": "", "tool_calls": []any{tcOf("c1", "", "{}")}},
		map[string]any{"role": "tool", "tool_call_id": "c1", "content": "result"},
		map[string]any{"role": "user", "content": "again"},
	}
	out := cleanMessages(in)
	if len(out) != 3 {
		t.Fatalf("should drop the call + its result (2 removed), got %d: %#v", len(out), out)
	}
	for _, m := range out {
		mm := m.(map[string]any)
		if mm["role"] == "assistant" {
			if _, ok := mm["tool_calls"]; ok {
				t.Fatalf("assistant tool_calls must be removed, got %#v", mm)
			}
		}
		if mm["role"] == "tool" {
			t.Fatalf("orphan tool result must not survive: %#v", mm)
		}
	}
	if out[0].(map[string]any)["content"] != "hi" || out[2].(map[string]any)["content"] != "again" {
		t.Fatalf("order/retention wrong: %#v", out)
	}
}

// TestCleanMessagesPartialDrop 一批两个调用都有结果，仅名字为空的那个删除，
// 干净的那个与其结果保留——验证按 id 对称裁剪，不误伤同批健康条目。
func TestCleanMessagesPartialDrop(t *testing.T) {
	in := []any{
		map[string]any{"role": "assistant", "tool_calls": []any{
			tcOf("c1", "", "{}"),
			tcOf("c2", "read", "{}"),
		}},
		map[string]any{"role": "tool", "tool_call_id": "c1", "content": "x"},
		map[string]any{"role": "tool", "tool_call_id": "c2", "content": "y"},
	}
	out := cleanMessages(in)
	if len(out) != 2 {
		t.Fatalf("should keep assistant + c2 result, drop c1 + its result: %d %#v", len(out), out)
	}
	asst := out[0].(map[string]any)
	tcs, ok := asst["tool_calls"].([]any)
	if !ok || len(tcs) != 1 {
		t.Fatalf("should keep exactly one call, got %#v", asst)
	}
	if id, _ := tcs[0].(map[string]any)["id"].(string); id != "c2" {
		t.Fatalf("kept call must be c2, got %s", id)
	}
	if id, _ := out[1].(map[string]any)["tool_call_id"].(string); id != "c2" {
		t.Fatalf("kept tool result must be c2, got %#v", out[1])
	}
}

// TestCleanMessagesWhitespaceNameIsDirty name 为纯空白按上游口径判脏。
func TestCleanMessagesWhitespaceNameIsDirty(t *testing.T) {
	in := []any{
		map[string]any{"role": "assistant", "tool_calls": []any{tcOf("c1", "   ", "{}")}},
		map[string]any{"role": "tool", "tool_call_id": "c1", "content": "x"},
	}
	out := cleanMessages(in)
	if len(out) != 0 && len(out) != 1 {
		t.Fatalf("unexpected len %d", len(out))
	}
	// assistant 应保留但无 tool_calls；tool 结果应删除
	if len(out) != 1 {
		t.Fatalf("only assistant should remain, got %#v", out)
	}
	if _, ok := out[0].(map[string]any)["tool_calls"]; ok {
		t.Fatalf("whitespace-name call must be dropped, tool_calls should be removed")
	}
}

// TestCleanMessagesHealthyNoOp 全干净时零改动、原样返回入参 slice（健康流量零拷贝）。
func TestCleanMessagesHealthyNoOp(t *testing.T) {
	in := []any{
		map[string]any{"role": "assistant", "tool_calls": []any{tcOf("c1", "read", "{}")}},
		map[string]any{"role": "tool", "tool_call_id": "c1", "content": "y"},
	}
	out := cleanMessages(in)
	if &out[0] != &in[0] {
		t.Fatal("healthy path must return the original slice, not a copy")
	}
	if !reflect.DeepEqual(out, in) {
		t.Fatal("healthy path must not mutate content")
	}
}

// TestCleanMessagesKeepsAssistantMultimodalContent 脏调用被删后，assistant 若带
// 多模态 content（数组）必须保留，不能被当空轮次误删。
func TestCleanMessagesKeepsAssistantMultimodalContent(t *testing.T) {
	in := []any{
		map[string]any{"role": "assistant",
			"content":    []any{map[string]any{"type": "text", "text": "here"}},
			"tool_calls": []any{tcOf("c1", "", "{}")}},
		map[string]any{"role": "tool", "tool_call_id": "c1", "content": "x"},
	}
	out := cleanMessages(in)
	if len(out) != 1 {
		t.Fatalf("assistant with multimodal content must survive, got %d", len(out))
	}
	if arr, ok := out[0].(map[string]any)["content"].([]any); !ok || len(arr) != 1 {
		t.Fatalf("multimodal content lost: %#v", out[0])
	}
}

// TestAppendToolCallDropsEmptyNameKeepsNoArg 聚合器写出闸门：空 name 丢弃；
// arguments 为空串的无参工具调用属合法载荷，必须保留（早期实现会误杀）。
func TestAppendToolCallDropsEmptyNameKeepsNoArg(t *testing.T) {
	var calls []any
	calls = appendToolCall(calls, tcOf("c1", "", "{}"))
	if len(calls) != 0 {
		t.Fatal("empty-name call must be dropped by the aggregate gate")
	}
	calls = appendToolCall(calls, tcOf("c2", "ping", ""))
	if len(calls) != 1 {
		t.Fatal("no-argument tool call (arguments==\"\") is valid and must be kept")
	}
	if id, _ := calls[0].(map[string]any)["id"].(string); id != "c2" {
		t.Fatalf("kept call must be c2, got %s", id)
	}
}
