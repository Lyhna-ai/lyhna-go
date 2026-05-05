package lyhna

import (
	"encoding/json"
	"testing"
)

func TestNewClient(t *testing.T) {
	c := NewClient("test_key", WithBaseURL("http://localhost:3000"))
	if c.apiKey != "test_key" {
		t.Errorf("expected apiKey=test_key, got %s", c.apiKey)
	}
	if c.baseURL != "http://localhost:3000" {
		t.Errorf("expected baseURL=http://localhost:3000, got %s", c.baseURL)
	}
}

func TestBindRequestWireShape(t *testing.T) {
	req := BindRequest{
		ActionType:    "deploy",
		Intent:        "release",
		IntentVersion: "1.0",
		Payload:       map[string]interface{}{"env": "prod"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	for _, key := range []string{"action_type", "intent", "intent_version", "action_payload"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected key %q present in wire shape, but was absent", key)
		}
	}

	for _, key := range []string{"authority_tier", "payload"} {
		if _, ok := m[key]; ok {
			t.Errorf("expected key %q absent from wire shape, but was present", key)
		}
	}
}

func TestReceiptFromMap(t *testing.T) {
	m := map[string]interface{}{
		"receipt_id":     "lrv2_123_abc",
		"outcome":        "APPROVED",
		"reason":         "all constraints satisfied",
		"action_type":    "deploy",
		"action_hash":    "abc123",
		"authority_tier": "tier_2",
		"timestamp":      "2026-04-13T00:00:00Z",
		"expires_at":     "2026-04-13T00:05:00Z",
		"tenant_hash":    "hash_abc",
		"signature":      "sig_xyz",
	}

	r := receiptFromMap(m)

	if r.ReceiptID != "lrv2_123_abc" {
		t.Errorf("ReceiptID = %q", r.ReceiptID)
	}
	if r.Outcome != "APPROVED" {
		t.Errorf("Outcome = %q", r.Outcome)
	}
	if r.OutcomeReason != "all constraints satisfied" {
		t.Errorf("OutcomeReason = %q", r.OutcomeReason)
	}
	if r.BoundAt != "2026-04-13T00:00:00Z" {
		t.Errorf("BoundAt = %q (should fall back to timestamp)", r.BoundAt)
	}
	if r.TenantID != "hash_abc" {
		t.Errorf("TenantID = %q (should fall back to tenant_hash)", r.TenantID)
	}
	if r.Raw == nil {
		t.Error("Raw should not be nil")
	}
}
