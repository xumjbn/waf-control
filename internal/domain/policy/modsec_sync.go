package policy

// modsec_sync.go —— 把 deploy/modsec/rules.d/<category>/<id>-<slug>.conf 解析成
// policies 表里的 builtin 规则。让前端『规则引擎』Tab 直接看到与 WAF agent
// 真正在执行的 ModSecurity 规则一一对应的条目。
//
// 规则源（按优先级）：
//   1) env WAF_MODSEC_RULES_DIR — 显式覆盖，便于 admin 临时挂别的目录
//   2) /etc/waf/modsec-rules    — docker-compose 容器内的卷挂载点
//   3) 其他几个 dev 路径
//   4) 编译期 embed 进二进制的 builtin_rules/（兜底，永远可用）
//
// 文件结构（约定，由 deploy/modsec/rules.d/_RULES.md 维护）：
//
//   # <category>/<id> — <human description>
//   SecRule <targets> "@<op> <pattern>" \
//       "id:<id>,\
//        phase:<n>,\
//        <deny|pass|drop|redirect>,\
//        status:<n>,\
//        log,\
//        msg:'<one-line message>',\
//        severity:'<CRITICAL|HIGH|MEDIUM|NOTICE>',\
//        tag:'<dotted-tag>'"
//
// SyncFromFS 是幂等的：用 modsec_id 做唯一键，已存在的就 UPDATE name/match/...
// 但保留 is_enabled（用户在 UI 里禁用过的不会被重新打开）。

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
)

// builtinRulesFS 把 deploy/modsec/rules.d 的镜像（waf-control/internal/domain/
// policy/builtin_rules/）打包进二进制。是最后一道兜底 —— 即便部署没挂载规则目录
// 也能保证规则页非空。
//
// 同步责任：Makefile `sync-builtin-rules` 把 deploy/modsec/rules.d → builtin_rules/
// 拉平；TestEmbeddedMatchesDeploy 在 CI 里防漂移。
//
//go:embed all:builtin_rules
var builtinRulesFS embed.FS

// ModsecRule 解析单个 .conf 文件后得到的结构化信息。
type ModsecRule struct {
	ModsecID    string // SecRule 里的 id:NNNNNN
	Category    string // 目录名 (sqli / xss / rce / bot / ...)
	Name        string // 「# <cat>/<id> — <desc>」里的 desc，缺失时回退 msg
	Description string // msg: 的内容
	Severity    string // critical→high / high→high / medium→medium / notice→low
	Action      string // deny→block / pass→log / drop→block / redirect→block
	Field       string // targets 转成 UI 可读串
	Match       string // 算子 + pattern（如 regex:(?i)union\s+select；截到 200 字符）
	Priority    int    // 取 modsec_id 前 N-3 位作排序基准，越小优先级越高
	Tag         string // tag: 的内容
}

var (
	rxFirstLine = regexp.MustCompile(`^#\s*([^/\s]+)/(\d+)\s*[—\-]\s*(.+)$`)
	rxID        = regexp.MustCompile(`id:(\d+)`)
	rxMsg       = regexp.MustCompile(`msg:'([^']+)'`)
	rxSeverity  = regexp.MustCompile(`severity:'([^']+)'`)
	rxTag       = regexp.MustCompile(`tag:'([^']+)'`)
	rxActionKW  = regexp.MustCompile(`(?:^|,|\s)(deny|pass|drop|redirect|allow|block)(?:,|\s|$)`)
	rxSecRule   = regexp.MustCompile(`(?s)SecRule\s+(\S+)\s+"(@\S+)\s+(.+?)"\s*\\?\s*"`)
)

// ParseBytes 解析单条规则的内容。category 来自文件所在子目录名。
func ParseBytes(category string, content []byte) (*ModsecRule, error) {
	src := string(content)
	out := &ModsecRule{Category: category}

	// 第一行：# <cat>/<id> — <desc>
	for _, line := range strings.SplitN(src, "\n", 2) {
		if m := rxFirstLine.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			out.Category = m[1]
			out.ModsecID = m[2]
			out.Name = strings.TrimSpace(m[3])
		}
		break
	}

	// 兜底从 SecRule 行抓 id
	if out.ModsecID == "" {
		if m := rxID.FindStringSubmatch(src); m != nil {
			out.ModsecID = m[1]
		}
	}
	if out.ModsecID == "" {
		return nil, errors.New("no modsec id found")
	}

	if m := rxMsg.FindStringSubmatch(src); m != nil {
		out.Description = m[1]
		if out.Name == "" {
			out.Name = m[1]
		}
	}
	if m := rxSeverity.FindStringSubmatch(src); m != nil {
		out.Severity = normalizeSeverity(m[1])
	} else {
		out.Severity = "medium"
	}
	if m := rxActionKW.FindStringSubmatch(src); m != nil {
		out.Action = normalizeAction(m[1])
	} else {
		out.Action = "log"
	}
	if m := rxTag.FindStringSubmatch(src); m != nil {
		out.Tag = m[1]
	}

	if m := rxSecRule.FindStringSubmatch(src); m != nil {
		out.Field = friendlyTargets(m[1])
		op := strings.TrimPrefix(m[2], "@")
		pattern := strings.TrimSpace(m[3])
		if len(pattern) > 200 {
			pattern = pattern[:200] + "…"
		}
		out.Match = op + ":" + pattern
	}

	if len(out.ModsecID) >= 4 {
		if v, err := strconv.Atoi(out.ModsecID[:len(out.ModsecID)-3]); err == nil {
			out.Priority = v
		}
	}
	if out.Priority <= 0 {
		out.Priority = 999
	}

	if out.Name == "" {
		out.Name = fmt.Sprintf("%s/%s", out.Category, out.ModsecID)
	}
	return out, nil
}

func normalizeSeverity(s string) string {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "CRITICAL", "HIGH":
		return "high"
	case "WARNING", "MEDIUM":
		return "medium"
	case "NOTICE", "INFO", "LOW":
		return "low"
	}
	return "medium"
}

func normalizeAction(a string) string {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "deny", "drop", "block":
		return "block"
	case "redirect":
		return "block"
	case "pass":
		return "log"
	case "allow":
		return "allow"
	}
	return "log"
}

func friendlyTargets(t string) string {
	parts := strings.Split(t, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		switch strings.ToUpper(strings.SplitN(p, ":", 2)[0]) {
		case "ARGS":
			out = append(out, "query")
		case "ARGS_POST", "REQUEST_BODY":
			out = append(out, "body")
		case "REQUEST_URI", "REQUEST_FILENAME":
			out = append(out, "uri")
		case "REQUEST_METHOD":
			out = append(out, "method")
		case "REQUEST_HEADERS":
			if idx := strings.Index(p, ":"); idx > 0 {
				out = append(out, "header."+p[idx+1:])
			} else {
				out = append(out, "header.*")
			}
		case "IP":
			out = append(out, "client.ip")
		default:
			out = append(out, strings.ToLower(p))
		}
	}
	seen := map[string]bool{}
	uniq := []string{}
	for _, x := range out {
		if !seen[x] {
			seen[x] = true
			uniq = append(uniq, x)
		}
	}
	r := strings.Join(uniq, "|")
	if len(r) > 64 {
		r = r[:64]
	}
	return r
}

// WalkFS 在任意 fs.FS 上扫描所有 .conf 文件并解析。
// root 是 FS 内部的子目录路径（embed 是 "builtin_rules"，os.DirFS(dir) 用 "."）。
func WalkFS(fsys fs.FS, root string) ([]ModsecRule, error) {
	var out []ModsecRule
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".conf") {
			return nil
		}
		// category = 上一层目录名
		category := path.Base(path.Dir(p))
		content, rerr := fs.ReadFile(fsys, p)
		if rerr != nil {
			slog.Warn("modsec rule read skipped", "file", p, "err", rerr)
			return nil
		}
		rule, perr := ParseBytes(category, content)
		if perr != nil {
			slog.Warn("modsec rule parse skipped", "file", p, "err", perr)
			return nil
		}
		out = append(out, *rule)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

// loadRules 按优先级（disk override → embed）拿规则。返回规则列表 + 数据源描述。
func loadRules() ([]ModsecRule, string, error) {
	if dir := os.Getenv("WAF_MODSEC_RULES_DIR"); dir != "" {
		if rules, err := WalkFS(os.DirFS(dir), "."); err == nil && len(rules) > 0 {
			return rules, "env:" + dir, nil
		} else if err != nil {
			slog.Warn("WAF_MODSEC_RULES_DIR walk failed", "dir", dir, "err", err)
		}
	}
	for _, p := range []string{
		"/etc/waf/modsec-rules",
		"../../deploy/modsec/rules.d",
		"../deploy/modsec/rules.d",
		"./deploy/modsec/rules.d",
	} {
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			if rules, err := WalkFS(os.DirFS(p), "."); err == nil && len(rules) > 0 {
				return rules, "disk:" + p, nil
			}
		}
	}
	// 最后兜底：编译期 embed 的副本。永远存在。
	rules, err := WalkFS(builtinRulesFS, "builtin_rules")
	if err != nil {
		return nil, "embed", fmt.Errorf("walk embed: %w", err)
	}
	return rules, "embed", nil
}

// SyncFromFS 把规则集合 upsert 进 policies 表（builtin=true）。返回 (inserted, updated, total, source)。
// 自动选择数据源（env / disk / embed）；调用方只要保证 DB 连通即可。
func (r *Repository) SyncFromFS(ctx context.Context) (int, int, int, string, error) {
	rules, source, err := loadRules()
	if err != nil {
		return 0, 0, 0, source, err
	}
	inserted, updated := 0, 0
	// 单条 INSERT ... ON CONFLICT 单 SQL —— 避免之前 SELECT-then-INSERT/UPDATE 的
	// TOCTOU（两个并发同步触发唯一约束冲突）。
	//
	// 关于覆盖策略：同步只覆盖『modsec 规则真正来源的字段』
	// （match_value/field/priority/category/severity/action），而**保留**用户在 UI
	// 修改过的 name/description 和 is_enabled。这通过 ON CONFLICT 时显式列出
	// 想覆盖的列实现。
	for _, m := range rules {
		tag, err := r.pool.Exec(ctx, `
			INSERT INTO policies
				(name, severity, action, is_enabled, description,
				 scope, field, match_value, priority, builtin, modsec_id, category)
			VALUES ($1,$2,$3,TRUE,$4,$5,$6,$7,$8,TRUE,$9,$10)
			ON CONFLICT (modsec_id) DO UPDATE
			   SET severity     = EXCLUDED.severity,
			       action       = EXCLUDED.action,
			       scope        = EXCLUDED.scope,
			       field        = EXCLUDED.field,
			       match_value  = EXCLUDED.match_value,
			       priority     = EXCLUDED.priority,
			       category     = EXCLUDED.category,
			       builtin      = TRUE,
			       updated_at   = NOW()
			 WHERE policies.modsec_id IS NOT NULL`,
			m.Name, m.Severity, m.Action, m.Description,
			"全部站点", m.Field, m.Match, m.Priority, m.ModsecID, m.Category)
		if err != nil {
			return inserted, updated, len(rules), source,
				fmt.Errorf("upsert builtin %s: %w", m.ModsecID, err)
		}
		// pg upsert 没法区分 INSERT/UPDATE，但 RowsAffected 1 等于成功，
		// 这里粗略按 'is the rule already there?' 计数 —— 不影响业务，只用于日志展示
		if tag.RowsAffected() > 0 {
			updated++ // 保守归 updated，启动期日志会显示『total=49 updated=49』
		}
	}
	// inserted 这里没法精确给出（PG ON CONFLICT 不区分），统一记 updated。
	// 真正『首次插入』的次数可以从 'INSERT 0 N' 的 N 减出，但代价大于价值。
	return inserted, updated, len(rules), source, nil
}
