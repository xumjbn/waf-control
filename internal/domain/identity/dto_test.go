package identity

import (
	"encoding/json"
	"strings"
	"testing"
)

// Wire-format tests that pin the JSON shape produced by ToDTO() so a
// regression here will fail loudly before the SPA ever sees a broken /me.

func TestRoleToDTOWildcardAdmin(t *testing.T) {
	role := Role{
		ID:          1,
		Name:        "系统管理员",
		RoleKey:     "system_admin",
		Permissions: []string{"*"},
		Color:       "#ef4444",
	}
	dto := role.ToDTO()
	if dto.Name != "system_admin" {
		t.Fatalf("DTO.name should be canonical key, got %q", dto.Name)
	}
	if dto.DisplayName != "系统管理员" {
		t.Fatalf("DTO.display_name should be Chinese, got %q", dto.DisplayName)
	}
	if !dto.Modules.Wildcard {
		t.Fatal("admin should produce wildcard modules")
	}
	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"modules":"*"`) {
		t.Fatalf("wildcard should serialise as \"*\", got %s", raw)
	}
}

func TestRoleToDTOScalarWildcardAdmin(t *testing.T) {
	// Migration 000008 stores the wildcard as scalar "*" rather than ["*"].
	// unmarshalPermissions normalises that to ["*"] internally; ToDTO must
	// still emit the scalar form on the wire.
	role := Role{
		ID:          2,
		Name:        "只读",
		RoleKey:     "readonly",
		Permissions: []string{"*"},
		Readonly:    true,
	}
	if !role.IsWildcard() {
		t.Fatal("readonly should be wildcard")
	}
	dto := role.ToDTO()
	if !dto.Modules.Wildcard {
		t.Fatal("DTO must mark wildcard")
	}
	if !dto.Readonly {
		t.Fatal("DTO must propagate readonly")
	}
}

func TestRoleToDTOExplicitList(t *testing.T) {
	role := Role{
		ID:          3,
		Name:        "审计员",
		RoleKey:     "auditor",
		Permissions: []string{"aggregation", "log", "report"},
		Readonly:    true,
		Color:       "#22d3ee",
	}
	dto := role.ToDTO()
	if dto.Modules.Wildcard {
		t.Fatal("auditor must not be wildcard")
	}
	if len(dto.Modules.List) != 3 {
		t.Fatalf("expected 3 modules, got %d", len(dto.Modules.List))
	}
	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `"modules":["aggregation","log","report"]`
	if !strings.Contains(string(raw), want) {
		t.Fatalf("expected %s in %s", want, raw)
	}
}

func TestUnmarshalPermissionsAcceptsBothShapes(t *testing.T) {
	cases := []struct {
		raw  string
		want []string
	}{
		{`"*"`, []string{"*"}},
		{`["*"]`, []string{"*"}},
		{`["site","log"]`, []string{"site", "log"}},
		{`[]`, nil},
	}
	for _, tc := range cases {
		var out []string
		if err := unmarshalPermissions([]byte(tc.raw), &out); err != nil {
			t.Fatalf("unmarshal %s: %v", tc.raw, err)
		}
		if len(out) != len(tc.want) {
			t.Fatalf("unmarshal %s: len mismatch %v vs %v", tc.raw, out, tc.want)
		}
		for i := range out {
			if out[i] != tc.want[i] {
				t.Fatalf("unmarshal %s: %v != %v", tc.raw, out, tc.want)
			}
		}
	}
}
