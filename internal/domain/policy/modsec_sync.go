package policy

// modsec_sync.go —— 把 deploy/modsec/rules.d/<category>/<id>-<slug>.conf 解析成
// policies 表里的 builtin 规则，让前端『规则引擎』Tab 直接能看到与 WAF agent
// 真正在执行的 ModSecurity 规则一一对应的条目。
//
// 目录约定：
//   deploy/modsec/rules.d/
//     sqli/120001-keyword-injection.conf
//     xss/130001-...
//     ...
//
// 文件结构（约定，由 deploy/modsec/_RULES.md 维护）：
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
// SyncFromDir 是幂等的：用 modsec_id 做唯一键，已存在的就 UPDATE name/match/...
// 但保留 is_enabled（用户在 UI 里禁用过的不会被重新打开）。

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

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
	Priority    int    // 取 modsec_id 前 5 位作排序基准，越小优先级越高
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

// ParseFile 解析单个 .conf 文件。失败返回 nil + 错误（调用方 skip）。
func ParseFile(category, path string) (*ModsecRule, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(raw)

	out := &ModsecRule{Category: category}

	// 第一行：# <cat>/<id> — <desc>
	for _, line := range strings.SplitN(content, "\n", 2) {
		if m := rxFirstLine.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			out.Category = m[1]
			out.ModsecID = m[2]
			out.Name = strings.TrimSpace(m[3])
		}
		break
	}

	// 兜底从 SecRule 行抓 id
	if out.ModsecID == "" {
		if m := rxID.FindStringSubmatch(content); m != nil {
			out.ModsecID = m[1]
		}
	}
	if out.ModsecID == "" {
		return nil, fmt.Errorf("no modsec id found in %s", path)
	}

	if m := rxMsg.FindStringSubmatch(content); m != nil {
		out.Description = m[1]
		if out.Name == "" {
			out.Name = m[1]
		}
	}
	if m := rxSeverity.FindStringSubmatch(content); m != nil {
		out.Severity = normalizeSeverity(m[1])
	} else {
		out.Severity = "medium"
	}
	if m := rxActionKW.FindStringSubmatch(content); m != nil {
		out.Action = normalizeAction(m[1])
	} else {
		out.Action = "log"
	}
	if m := rxTag.FindStringSubmatch(content); m != nil {
		out.Tag = m[1]
	}

	// SecRule <targets> "@<op> <pattern>"
	if m := rxSecRule.FindStringSubmatch(content); m != nil {
		out.Field = friendlyTargets(m[1])
		op := strings.TrimPrefix(m[2], "@")
		pattern := strings.TrimSpace(m[3])
		if len(pattern) > 200 {
			pattern = pattern[:200] + "…"
		}
		out.Match = op + ":" + pattern
	}

	// modsec_id 前 4-5 位决定排序：120001 → 120
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

// friendlyTargets 把 ModSecurity 的 targets (ARGS|REQUEST_BODY|REQUEST_HEADERS:Cookie)
// 翻译成 UI 习惯的 body|query / header.X / uri 之类的串。
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
	// 去重 + 限长
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

// WalkRulesDir 扫描整棵规则树，返回所有解析成功的规则。失败的文件只打 warn。
func WalkRulesDir(root string) ([]ModsecRule, error) {
	if root == "" {
		return nil, errors.New("modsec rules dir is empty")
	}
	stat, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("stat %s: %w", root, err)
	}
	if !stat.IsDir() {
		return nil, fmt.Errorf("%s is not a dir", root)
	}

	var out []ModsecRule
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".conf") {
			return nil
		}
		// category = 上一层目录名
		category := filepath.Base(filepath.Dir(path))
		rule, perr := ParseFile(category, path)
		if perr != nil {
			slog.Warn("modsec rule parse skipped", "file", path, "err", perr)
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

// SyncFromDir 把 root 下解析到的规则 upsert 进 policies 表（builtin=true）。
// 返回 (inserted, updated, total)。已存在的（按 modsec_id）只更新 name/severity/
// action/scope/field/match/priority/description，保留 is_enabled。
func (r *Repository) SyncFromDir(ctx context.Context, root string) (int, int, int, error) {
	rules, err := WalkRulesDir(root)
	if err != nil {
		return 0, 0, 0, err
	}
	inserted, updated := 0, 0
	for _, m := range rules {
		var (
			existingID int64
			existing   bool
		)
		err := r.pool.QueryRow(ctx,
			`SELECT id FROM policies WHERE modsec_id = $1`, m.ModsecID,
		).Scan(&existingID)
		if err == nil {
			existing = true
		}

		if existing {
			_, err := r.pool.Exec(ctx, `
				UPDATE policies
				   SET name=$1, severity=$2, action=$3, description=$4,
				       scope=$5, field=$6, match_value=$7, priority=$8,
				       builtin=TRUE, updated_at=NOW()
				 WHERE id=$9`,
				m.Name, m.Severity, m.Action, m.Description,
				"全部站点", m.Field, m.Match, m.Priority, existingID)
			if err != nil {
				return inserted, updated, len(rules), fmt.Errorf("update builtin %s: %w", m.ModsecID, err)
			}
			updated++
		} else {
			_, err := r.pool.Exec(ctx, `
				INSERT INTO policies
					(name, severity, action, is_enabled, description,
					 scope, field, match_value, priority, builtin, modsec_id)
				VALUES ($1,$2,$3,TRUE,$4,$5,$6,$7,$8,TRUE,$9)`,
				m.Name, m.Severity, m.Action, m.Description,
				"全部站点", m.Field, m.Match, m.Priority, m.ModsecID)
			if err != nil {
				return inserted, updated, len(rules), fmt.Errorf("insert builtin %s: %w", m.ModsecID, err)
			}
			inserted++
		}
	}
	return inserted, updated, len(rules), nil
}
