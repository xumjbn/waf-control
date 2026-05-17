package deploy

import (
	"strings"
	"testing"
)

func TestGenerateNginx(t *testing.T) {
	site := &SiteConfig{
		Domain:    "example.com",
		Protocol:  "https",
		Upstreams: []string{"10.0.0.1:8080", "10.0.0.2:8080"},
		SSLName:   "example.com",
	}

	cfg := generateNginx(site)

	if !strings.Contains(cfg, "server_name example.com") {
		t.Error("missing server_name")
	}
	if !strings.Contains(cfg, "listen 443 ssl http2") {
		t.Error("missing https listen")
	}
	if !strings.Contains(cfg, "ssl_certificate") {
		t.Error("missing ssl cert")
	}
	if !strings.Contains(cfg, "proxy_pass") {
		t.Error("missing proxy_pass")
	}
	if !strings.Contains(cfg, "modsecurity on") {
		t.Error("missing modsecurity")
	}
	if !strings.Contains(cfg, "upstream backend_example_com") {
		t.Error("missing upstream")
	}
	for _, u := range site.Upstreams {
		if !strings.Contains(cfg, u) {
			t.Errorf("missing upstream server %s", u)
		}
	}
}

func TestGenerateNginxHTTP(t *testing.T) {
	site := &SiteConfig{
		Domain:   "example.com",
		Protocol: "http",
	}

	cfg := generateNginx(site)

	if strings.Contains(cfg, "ssl_") {
		t.Error("http config should not contain ssl directives")
	}
	if !strings.Contains(cfg, "listen 80") {
		t.Error("missing port 80 listen")
	}
	if !strings.Contains(cfg, "modsecurity on") {
		t.Error("missing modsecurity")
	}
}

func TestGenerateModsecOn(t *testing.T) {
	policy := &PolicyConfig{
		Name: "strict-policy",
		Mode: "on",
		RuleGroups: []RuleGroup{
			{
				Category: "SQL Injection",
				Rules: []ModsecRule{
					{ID: 100001, Phase: 2, Severity: "CRITICAL", Message: "SQL Injection", Match: "ARGS", Operator: "rx", Pattern: "(?:')|(?:--)|(/\\*.*\\*/)", Action: "block"},
					{ID: 100002, Phase: 1, Severity: "CRITICAL", Message: "SQL Comment", Match: "ARGS", Operator: "rx", Pattern: "(?i)union.*select", Action: "block"},
				},
			},
			{
				Category: "XSS",
				Rules: []ModsecRule{
					{ID: 200001, Phase: 2, Severity: "CRITICAL", Message: "XSS - script tag", Match: "ARGS", Operator: "rx", Pattern: "(?i)<script[^>]*>", Action: "block"},
				},
			},
		},
	}

	cfg := generateModsec(policy)

	if !strings.Contains(cfg, "SecRuleEngine On") {
		t.Error("expected SecRuleEngine On")
	}
	if !strings.Contains(cfg, "policy: strict-policy") {
		t.Error("missing policy name")
	}
	if !strings.Contains(cfg, "100001") {
		t.Error("missing rule 100001")
	}
	if !strings.Contains(cfg, "200001") {
		t.Error("missing rule 200001")
	}
	if !strings.Contains(cfg, "SQL Injection") {
		t.Error("missing rule group comment")
	}
}

func TestGenerateModsecDetection(t *testing.T) {
	policy := &PolicyConfig{
		Name: "audit-policy",
		Mode: "detection",
	}

	cfg := generateModsec(policy)

	if !strings.Contains(cfg, "SecRuleEngine DetectionOnly") {
		t.Error("expected DetectionOnly mode")
	}
	if !strings.Contains(cfg, "SecDefaultAction") {
		t.Error("expected SecDefaultAction for detection mode")
	}
}

func TestGenerateModsecOff(t *testing.T) {
	policy := &PolicyConfig{
		Name: "bypass",
		Mode: "off",
	}

	cfg := generateModsec(policy)

	if !strings.Contains(cfg, "SecRuleEngine Off") {
		t.Error("expected SecRuleEngine Off")
	}
}

func TestFormatRule(t *testing.T) {
	rule := ModsecRule{
		ID:       300001,
		Phase:    2,
		Severity: "CRITICAL",
		Message:  "Test Rule",
		Match:    "ARGS:foo",
		Operator: "rx",
		Pattern:  "badpattern",
		Action:   "block",
	}

	result := formatRule(rule)

	if !strings.HasPrefix(result, "SecRule") {
		t.Error("should start with SecRule")
	}
	if !strings.Contains(result, "ARGS:foo") {
		t.Error("missing match variable")
	}
	if !strings.Contains(result, "300001") {
		t.Error("missing rule ID")
	}
	if !strings.Contains(result, "severity:'CRITICAL'") {
		t.Error("missing severity")
	}
}

