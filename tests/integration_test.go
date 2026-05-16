package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/waf-control/internal/domain/acl"
	"github.com/waf-control/internal/domain/ha"
	"github.com/waf-control/internal/domain/loadbalance"
	"github.com/waf-control/internal/domain/logs"
	"github.com/waf-control/internal/domain/system"
)

func setupRouter(pool testPool) chi.Router {
	r := chi.NewRouter()
	loadbalance.RegisterRoutes(r, pool.Pool())
	acl.RegisterRoutes(r, pool.Pool())
	ha.RegisterRoutes(r, pool.Pool())
	system.RegisterRoutes(r, pool.Pool())
	logs.RegisterRoutes(r, pool.Pool())
	return r
}

func TestACLRoutes(t *testing.T) {
	pool := getTestPool(t)
	r := setupRouter(pool)

	t.Run("create acl rule", func(t *testing.T) {
		body := map[string]interface{}{
			"name":      "block-ssh",
			"direction": "inbound",
			"action":    "deny",
			"protocol":  "tcp",
			"dst_port":  22,
			"priority":  10,
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/acl/rules", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}

		var rule map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &rule)
		if rule["name"] != "block-ssh" {
			t.Fatalf("expected name block-ssh, got %v", rule["name"])
		}
	})

	t.Run("list acl rules", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/acl/rules", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("create acl rule missing name", func(t *testing.T) {
		body := map[string]interface{}{"action": "deny"}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/acl/rules", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", w.Code)
		}
	})
}

func TestLoadBalanceRoutes(t *testing.T) {
	pool := getTestPool(t)
	r := setupRouter(pool)

	var vipID float64

	t.Run("create vip", func(t *testing.T) {
		body := map[string]interface{}{
			"name":        "web-vip",
			"address":     "192.168.1.100",
			"port":        443,
			"protocol":    "tcp",
			"lb_method":   "round_robin",
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/lb/vips", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
		}
		var resp map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &resp)
		vipID = resp["id"].(float64)
	})

	t.Run("list vips", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/lb/vips", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("delete vip", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/lb/vips/%d", int64(vipID)), nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestSystemSettingsRoutes(t *testing.T) {
	pool := getTestPool(t)
	r := setupRouter(pool)

	t.Run("upsert setting", func(t *testing.T) {
		body := map[string]interface{}{
			"key":      "waf_mode",
			"value":    "detect",
			"category": "general",
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, "/system/settings", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("list settings", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/system/settings?category=general", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestHARoutes(t *testing.T) {
	pool := getTestPool(t)
	r := setupRouter(pool)

	t.Run("upsert ha config", func(t *testing.T) {
		body := map[string]interface{}{
			"mode":          "active-standby",
			"virtual_ip":    "192.168.1.200",
			"priority":      100,
			"interface":     "eth0",
			"peer_address":  "192.168.1.2",
			"heartbeat_sec": 5,
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPut, "/ha/config", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
		}
	})

	t.Run("get ha config", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/ha/config", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}

func TestLogRoutes(t *testing.T) {
	pool := getTestPool(t)
	r := setupRouter(pool)

	t.Run("list attack logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/logs/attack?page=1&page_size=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("list antivirus logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/logs/antivirus?page=1&page_size=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})

	t.Run("list antitamper logs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/logs/antitamper?page=1&page_size=10", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", w.Code)
		}
	})
}
