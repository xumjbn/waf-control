package policy

import "testing"

func TestEvalRule_Regex(t *testing.T) {
	r := DryRunRule{
		Field:  "body|query",
		Match:  `regex:(?i)union\s+select`,
		Action: "block",
	}
	req := DryRunRequest{
		Method: "POST",
		URL:    "/login?next=/admin",
		Body:   "username=admin&password=' UNION SELECT 1,2--",
	}
	res := evalRule(r, req)
	if !res.Matched {
		t.Fatalf("expected match, got %+v", res)
	}
	if res.Action != "block" {
		t.Errorf("expected action=block, got %q", res.Action)
	}
	if len(res.HitFields) == 0 {
		t.Errorf("expected hit fields, got empty")
	}
}

func TestEvalRule_Contains_Miss(t *testing.T) {
	r := DryRunRule{
		Field:  "uri",
		Match:  "contains:/admin",
		Action: "challenge",
	}
	req := DryRunRequest{URL: "/api/v1/users/123"}
	res := evalRule(r, req)
	if res.Matched {
		t.Errorf("expected miss, got matched")
	}
	if res.Action != "pass" {
		t.Errorf("on miss, action should be 'pass', got %q", res.Action)
	}
}

func TestEvalRule_HeaderCaseInsensitive(t *testing.T) {
	r := DryRunRule{
		Field:  "header.UA",
		Match:  "regex:(?i)sqlmap",
		Action: "block",
	}
	req := DryRunRequest{
		Headers: map[string]string{"user-agent": "sqlmap/1.6"},
	}
	res := evalRule(r, req)
	if !res.Matched {
		t.Errorf("expected match (header case insensitive), got %+v", res)
	}
}

func TestEvalRule_InList(t *testing.T) {
	r := DryRunRule{
		Field:  "method",
		Match:  "in:DELETE,PUT,PATCH",
		Action: "block",
	}
	if !evalRule(r, DryRunRequest{Method: "DELETE"}).Matched {
		t.Error("DELETE should match")
	}
	if evalRule(r, DryRunRequest{Method: "GET"}).Matched {
		t.Error("GET should not match")
	}
}
