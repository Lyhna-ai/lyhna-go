package lyhna

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

// TestHostedSmoke mirrors the Phase 10 Gate 1 JS smoke (scripts/smoke-hosted.ts).
// It requires LYHNA_HOSTED_BASE_URL (e.g. https://host/v1) and is skipped when unset.
//
// Sequence: signup → bind (via SDK) → receipts/count → receipts list (hard match).
func TestHostedSmoke(t *testing.T) {
	baseURL := strings.TrimRight(os.Getenv("LYHNA_HOSTED_BASE_URL"), "/")
	if baseURL == "" {
		t.Skip("LYHNA_HOSTED_BASE_URL not set; skipping hosted smoke")
	}

	// The SDK appends /v1/bind itself; strip /v1 from the env-var base for the SDK.
	// Raw HTTP calls use baseURL directly (which includes /v1).
	sdkBase := baseURL
	if strings.HasSuffix(sdkBase, "/v1") {
		sdkBase = strings.TrimSuffix(sdkBase, "/v1")
	}

	ts := time.Now().UnixMilli()
	emailSuffix := smokeRandHex(3)
	email := fmt.Sprintf("smoke+%d-%s@lyhna.dev", ts, emailSuffix)
	password := fmt.Sprintf("SmokeGo-%s", smokeRandHex(8))
	nonce := smokeRandHex(8)

	t.Logf("[A] email: %s", email)

	// ── B. Signup ──────────────────────────────────────────────────────────────
	t.Log("[B] POST /auth/signup ...")

	signupPayload, _ := json.Marshal(map[string]string{"email": email, "password": password})
	signupReq, _ := http.NewRequest("POST", baseURL+"/auth/signup", bytes.NewReader(signupPayload))
	signupReq.Header.Set("Content-Type", "application/json")

	signupResp, err := http.DefaultClient.Do(signupReq)
	if err != nil {
		t.Fatalf("POST /auth/signup unreachable: %v", err)
	}
	defer signupResp.Body.Close()

	signupRaw, _ := io.ReadAll(signupResp.Body)
	if signupResp.StatusCode != 201 {
		t.Fatalf("POST /auth/signup → HTTP %d (expected 201): %s", signupResp.StatusCode, signupRaw)
	}

	var signupData map[string]interface{}
	if err := json.Unmarshal(signupRaw, &signupData); err != nil {
		t.Fatalf("POST /auth/signup response not valid JSON: %v", err)
	}

	tenantID, _ := signupData["tenant_id"].(string)
	if !strings.HasPrefix(tenantID, "tenant_") {
		t.Fatalf("tenant_id malformed or missing: %q", tenantID)
	}
	apiKey, _ := signupData["api_key"].(string)
	if !strings.HasPrefix(apiKey, "lyhna_") {
		t.Fatalf("api_key malformed or missing")
	}
	if signupData["api_key_warning"] == nil {
		t.Fatalf("api_key_warning missing from signup response")
	}

	var sessionToken string
	for _, c := range signupResp.Cookies() {
		if c.Name == "lyhna_session" {
			sessionToken = c.Value
			break
		}
	}
	if sessionToken == "" {
		t.Fatalf("lyhna_session cookie not found in signup response")
	}

	t.Logf("  ✓ signup ok (tenant_id=%s)", tenantID)

	// ── C. Bind via SDK ────────────────────────────────────────────────────────
	t.Log("[C] Bind via SDK ...")

	client := NewClient(apiKey, WithBaseURL(sdkBase))
	receipt, err := client.Bind(context.Background(), BindRequest{
		ActionType:    "smoke_go_hosted_bind",
		Intent:        "phase_4e_go_sdk_smoke",
		IntentVersion: "1.0.0",
		Payload: map[string]interface{}{
			"source": "phase_4e_go_sdk",
			"nonce":  nonce,
		},
	})
	if err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	version, _ := receipt.Raw["version"].(string)
	if version != "LYHNA_RECEIPT_V2" {
		t.Fatalf("receipt.version: expected LYHNA_RECEIPT_V2, got %q", version)
	}
	if !strings.HasPrefix(receipt.ReceiptID, "lrv2_") {
		t.Fatalf("receipt_id malformed: %q", receipt.ReceiptID)
	}
	if receipt.CanonicalHash == "" {
		t.Fatalf("receipt.canonical_hash empty")
	}
	if receipt.Signature == "" {
		t.Fatalf("receipt.signature empty")
	}
	if pubKey, _ := receipt.Raw["public_key"].(string); pubKey == "" {
		t.Fatalf("receipt.public_key empty")
	}
	switch receipt.Outcome {
	case "APPROVED", "REFUSED", "ESCALATED":
	default:
		t.Fatalf("receipt.outcome unexpected: %q", receipt.Outcome)
	}

	t.Logf("  ✓ bind ok (receipt_id=%s outcome=%s)", receipt.ReceiptID, receipt.Outcome)

	// ── D. Receipt count ───────────────────────────────────────────────────────
	t.Log("[D] GET /auth/receipts/count ...")

	countReq, _ := http.NewRequest("GET", baseURL+"/auth/receipts/count", nil)
	countReq.AddCookie(&http.Cookie{Name: "lyhna_session", Value: sessionToken})

	countResp, err := http.DefaultClient.Do(countReq)
	if err != nil {
		t.Fatalf("GET /auth/receipts/count failed: %v", err)
	}
	defer countResp.Body.Close()

	countRaw, _ := io.ReadAll(countResp.Body)
	if countResp.StatusCode != 200 {
		t.Fatalf("GET /auth/receipts/count → HTTP %d (expected 200): %s", countResp.StatusCode, countRaw)
	}

	var countData map[string]interface{}
	if err := json.Unmarshal(countRaw, &countData); err != nil {
		t.Fatalf("GET /auth/receipts/count not valid JSON: %v", err)
	}
	cnt, ok := countData["count"].(float64)
	if !ok {
		t.Fatalf("count field missing or not a number: %v", countData)
	}
	if cnt < 1 {
		t.Fatalf("receipt count is %.0f after bind — expected >= 1", cnt)
	}

	t.Logf("  ✓ receipt count ok (count=%.0f)", cnt)

	// ── E. Receipt list — hard receipt_id match ────────────────────────────────
	t.Log("[E] GET /auth/receipts ...")

	listReq, _ := http.NewRequest("GET", baseURL+"/auth/receipts", nil)
	listReq.AddCookie(&http.Cookie{Name: "lyhna_session", Value: sessionToken})

	listResp, err := http.DefaultClient.Do(listReq)
	if err != nil {
		t.Fatalf("GET /auth/receipts failed: %v", err)
	}
	defer listResp.Body.Close()

	listRaw, _ := io.ReadAll(listResp.Body)
	if listResp.StatusCode != 200 {
		t.Fatalf("GET /auth/receipts → HTTP %d (expected 200): %s", listResp.StatusCode, listRaw)
	}

	var listData map[string]interface{}
	if err := json.Unmarshal(listRaw, &listData); err != nil {
		t.Fatalf("GET /auth/receipts not valid JSON: %v", err)
	}

	receipts, ok := listData["receipts"].([]interface{})
	if !ok {
		ks := make([]string, 0, len(listData))
		for k := range listData {
			ks = append(ks, k)
		}
		t.Fatalf("receipts field missing or not an array; top-level keys: %v", ks)
	}

	found := false
	for _, entry := range receipts {
		if m, ok := entry.(map[string]interface{}); ok {
			if m["receipt_id"] == receipt.ReceiptID {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("receipt_id %s not found in /auth/receipts list", receipt.ReceiptID)
	}

	t.Logf("  ✓ receipt list match ok (receipt_id=%s)", receipt.ReceiptID)
	t.Logf("  tenant_id=%s", tenantID)
}

func smokeRandHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("crypto/rand.Read: %v", err))
	}
	return hex.EncodeToString(b)
}
