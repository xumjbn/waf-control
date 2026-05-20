package policy

import (
	"path/filepath"
	"testing"
)

// 验证解析器能正确处理 deploy/modsec/rules.d/ 下的实际规则文件。
// 不连 DB，只 parse。
func TestParseRealRules(t *testing.T) {
	root, err := filepath.Abs("../../../../deploy/modsec/rules.d")
	if err != nil {
		t.Fatal(err)
	}
	rules, err := WalkRulesDir(root)
	if err != nil {
		t.Skipf("rules dir not found: %v", err)
		return
	}
	if len(rules) == 0 {
		t.Fatalf("expected to parse some rules from %s, got 0", root)
	}
	t.Logf("parsed %d rules", len(rules))

	// 抽样三个典型分类，验证字段都填上了。
	seenCats := map[string]bool{}
	for _, r := range rules {
		if r.ModsecID == "" {
			t.Errorf("rule with empty modsec_id: name=%q file-cat=%q", r.Name, r.Category)
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
			t.Errorf("missing category %q in parsed rules; got %v", want, seenCats)
		}
	}
}
