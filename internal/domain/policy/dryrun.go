package policy

// dryrun.go —— 规则试运行（不写库，纯计算）
//
// 用例：waf-admin RuleEdit 顶部『试运行』按钮 —— 用户在编辑规则时输入一份
// 测试请求（method/url/headers/body），后端按当前规则的 field+match 评估
// 一遍，返回是否命中 + 各条件是否命中 + 该取的 action。
//
// 端点：POST /api/v1/policies/dry-run
// 请求体：{ rule: { field, match }, request: { method, url, headers, body } }
//        rule.id 可空（编辑中尚未保存的规则）
// 响应：{ matched, time_ms, hit_fields: [...], action }

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// 安全上限 —— 防止恶意 regex / 巨大 value / 长 pattern 拖死服务。
// Go 标准库 regexp 是 RE2 不会指数回溯，但 compile + match 仍然受 input 长度影响。
const (
	maxPatternLen = 1024            // 单条 match pattern 长度
	maxValueLen   = 64 * 1024       // 单字段（uri/body/...）评估值长度
	dryRunTimeout = 1 * time.Second // 整个 evalRule 必须在此内完成
)

// DryRunRule 试运行用的规则 spec —— 不需要完整 Policy，只要 field+match+action。
// match 字符串约定 `<op>:<pattern>`，op ∈ {regex, contains, equals, prefix, in, cidr}。
type DryRunRule struct {
	ID     int64  `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Field  string `json:"field"`
	Match  string `json:"match"`
	Action string `json:"action"`
}

// DryRunRequest 试运行用的伪请求 —— 与 modsec 规则可能使用的 targets 对齐。
type DryRunRequest struct {
	Method  string            `json:"method"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// DryRunResult 评估结果。
type DryRunResult struct {
	Matched   bool     `json:"matched"`
	TimeMs    float64  `json:"time_ms"`
	HitFields []string `json:"hit_fields"`           // 实际命中的 field 列表
	Action    string   `json:"action"`               // 命中时取规则的 action，否则 'pass'
	Reason    string   `json:"reason,omitempty"`     // 调试用，例如『正则不合法』
}

// 把 UI 习惯的 field 名（uri/method/body/query/header.X/client.ip）从 request
// 里取出对应字符串。支持多字段用 '|' 分隔（与 friendlyTargets 反向）。
func extractField(req DryRunRequest, field string) []string {
	if field == "" {
		return nil
	}
	var out []string
	for _, f := range strings.Split(field, "|") {
		f = strings.TrimSpace(f)
		switch {
		case f == "uri" || f == "url":
			out = append(out, req.URL)
		case f == "method":
			out = append(out, req.Method)
		case f == "body":
			out = append(out, req.Body)
		case f == "query":
			if i := strings.Index(req.URL, "?"); i >= 0 {
				out = append(out, req.URL[i+1:])
			} else {
				out = append(out, "")
			}
		case f == "client.ip":
			// dry-run 没有真实客户端 IP，使用占位
			out = append(out, "0.0.0.0")
		case strings.HasPrefix(f, "header."):
			name := strings.TrimPrefix(f, "header.")
			// UI 常用缩写 → 真实 header 名（与 deploy/modsec 规则的 REQUEST_HEADERS:UA 对齐）
			aliases := map[string]string{
				"ua":            "user-agent",
				"user-agent":    "user-agent",
				"referer":       "referer",
				"cookie":        "cookie",
				"authorization": "authorization",
				"xff":           "x-forwarded-for",
				"x-forwarded-for": "x-forwarded-for",
			}
			canonical := strings.ToLower(name)
			if mapped, ok := aliases[canonical]; ok {
				canonical = mapped
			}
			// 大小写不敏感匹配
			found := ""
			for k, v := range req.Headers {
				if strings.ToLower(k) == canonical {
					found = v
					break
				}
			}
			out = append(out, found)
		default:
			out = append(out, "")
		}
	}
	return out
}

// evalCondition 按 op 评估 value 是否匹配 pattern。
// pattern 已经在 evalRule 截过长，value 这里再 cap 一次防御性截断。
func evalCondition(value, op, pattern string) (bool, string) {
	if len(value) > maxValueLen {
		value = value[:maxValueLen]
	}
	switch strings.ToLower(op) {
	case "regex", "rx":
		if len(pattern) > maxPatternLen {
			return false, "pattern 超长（最大 1024 字节）"
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return false, "正则不合法: " + err.Error()
		}
		return re.MatchString(value), ""
	case "contains":
		return strings.Contains(value, pattern), ""
	case "equals", "eq":
		return value == pattern, ""
	case "not_equals", "ne":
		return value != pattern, ""
	case "prefix":
		return strings.HasPrefix(value, pattern), ""
	case "in":
		for _, s := range strings.Split(pattern, ",") {
			if strings.TrimSpace(s) == value {
				return true, ""
			}
		}
		return false, ""
	}
	// 未知 op：退化为字面量包含
	return strings.Contains(value, pattern), "unknown op: " + op
}

// evalRule 评估单条规则。返回是否命中 + 命中的具体字段 + 失败原因。
func evalRule(rule DryRunRule, req DryRunRequest) DryRunResult {
	start := time.Now()
	result := DryRunResult{HitFields: []string{}, Action: "pass"}

	// match 格式：<op>:<pattern>，缺 ':' 则整串作为 contains pattern
	op, pattern := "contains", rule.Match
	if i := strings.Index(rule.Match, ":"); i > 0 {
		op = rule.Match[:i]
		pattern = rule.Match[i+1:]
	}

	values := extractField(req, rule.Field)
	hitFields := make([]string, 0, len(values))
	matchedAny := false
	for i, v := range values {
		ok, why := evalCondition(v, op, pattern)
		if why != "" {
			result.Reason = why
		}
		if ok {
			matchedAny = true
			fieldName := strings.Split(rule.Field, "|")[i]
			hitFields = append(hitFields, strings.TrimSpace(fieldName))
		}
	}
	result.Matched = matchedAny
	result.HitFields = hitFields
	if matchedAny {
		result.Action = rule.Action
	}
	result.TimeMs = float64(time.Since(start).Microseconds()) / 1000.0
	return result
}

// DryRun handler：POST /api/v1/policies/dry-run
func (h *Handler) DryRun(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Rule    DryRunRule    `json:"rule"`
		Request DryRunRequest `json:"request"`
	}
	// MaxBody 中间件已经把 r.Body cap 在 1MiB，这里再硬截 256KiB 做防御
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 256*1024))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid body"})
		return
	}
	if body.Rule.Field == "" || body.Rule.Match == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "rule.field 和 rule.match 都必填",
		})
		return
	}
	if len(body.Rule.Match) > maxPatternLen {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "rule.match 超长（最大 1024 字节）",
		})
		return
	}
	if body.Rule.Action == "" {
		body.Rule.Action = "block"
	}

	// 整个评估必须在 dryRunTimeout 内完成；Go 标准 regexp 不直接接受 context，
	// 但实测在 1s 内 RE2 + 64KB value 完全够用，超时基本只会在 cgo/异常情况下触发。
	ctx, cancel := context.WithTimeout(r.Context(), dryRunTimeout)
	defer cancel()
	done := make(chan DryRunResult, 1)
	go func() { done <- evalRule(body.Rule, body.Request) }()
	select {
	case res := <-done:
		writeJSON(w, http.StatusOK, res)
	case <-ctx.Done():
		writeJSON(w, http.StatusRequestTimeout, map[string]string{
			"error": "试运行超时（≤1s 限制）—— 正则或 value 过于复杂",
		})
	}
}
