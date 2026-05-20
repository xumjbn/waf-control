package policy

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseEmbed 验证编译期 embed 的规则都能解析。
// 这个测试不依赖外部 deploy/ 路径，CI 上永远跑得过。
func TestParseEmbed(t *testing.T) {
	rules, err := WalkFS(builtinRulesFS, "builtin_rules")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) < 30 {
		t.Fatalf("expected >=30 rules embedded, got %d", len(rules))
	}
	t.Logf("embed parsed %d rules", len(rules))

	seenCats := map[string]bool{}
	for _, r := range rules {
		if r.ModsecID == "" {
			t.Errorf("rule with empty modsec_id: name=%q cat=%q", r.Name, r.Category)
		}
		if r.Name == "" {
			t.Errorf("rule %s has empty name", r.ModsecID)
		}
		if r.Action == "" {
			t.Errorf("rule %s has empty action", r.ModsecID)
		}
		if r.Severity == "" {
			t.Errorf("rule %s has empty severity", r.ModsecID)
		}
		seenCats[r.Category] = true
	}
	for _, want := range []string{"sqli", "xss", "bot"} {
		if !seenCats[want] {
			t.Errorf("missing category %q; got %v", want, seenCats)
		}
	}
}

// TestEmbeddedMatchesDeploy 在 dev 机上跑（CI 上若 deploy/ 路径不存在则 skip）。
// 防止 waf-control 内 builtin_rules/ 与 deploy/modsec/rules.d/ 漂移。
// 通过 `make sync-builtin-rules`（OpenWAF 根 Makefile）保持同步。
func TestEmbeddedMatchesDeploy(t *testing.T) {
	deployRoot, err := filepath.Abs("../../../../deploy/modsec/rules.d")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(deployRoot); err != nil {
		t.Skipf("deploy rules not present (likely CI without monorepo root): %v", err)
		return
	}

	want := map[string]string{}
	if err := filepath.WalkDir(deployRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".conf") {
			return err
		}
		rel, _ := filepath.Rel(deployRoot, p)
		body, _ := os.ReadFile(p)
		want[filepath.ToSlash(rel)] = string(body)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	got := map[string]string{}
	if err := fs.WalkDir(builtinRulesFS, "builtin_rules", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".conf") {
			return err
		}
		body, _ := fs.ReadFile(builtinRulesFS, p)
		rel := strings.TrimPrefix(p, "builtin_rules/")
		got[rel] = string(body)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	for k, w := range want {
		g, ok := got[k]
		if !ok {
			t.Errorf("file %s present in deploy/ but missing in embed/. 运行 make sync-builtin-rules", k)
			continue
		}
		if g != w {
			t.Errorf("file %s diverged between deploy/ and embed/. 运行 make sync-builtin-rules", k)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("file %s present in embed/ but missing in deploy/.  规则已废弃？运行 make sync-builtin-rules", k)
		}
	}
}
