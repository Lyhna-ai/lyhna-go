package lyhna

// BindRequest contains the parameters for a bind call.
type BindRequest struct {
	ActionType    string                 `json:"action_type"`
	Intent        string                 `json:"intent"`
	IntentVersion string                 `json:"intent_version"`
	Payload       map[string]interface{} `json:"action_payload,omitempty"`
}

// Receipt represents a LYHNA_RECEIPT_V2.
type Receipt struct {
	ReceiptID     string `json:"receipt_id"`
	Outcome       string `json:"outcome"`
	OutcomeReason string `json:"outcome_reason"`
	EscalateTo    string `json:"escalate_to"`
	ActionType    string `json:"action_type"`
	ActionHash    string `json:"action_hash"`
	AuthorityTier string `json:"authority_tier"`
	BoundAt       string `json:"bound_at"`
	ExpiresAt     string `json:"expires_at"`
	TenantID      string `json:"tenant_id"`
	CanonicalHash string `json:"canonical_hash"`
	Signature     string `json:"signature"`

	// Raw holds the complete API response for fields not explicitly typed.
	Raw map[string]interface{} `json:"-"`
}

// VerifyResult contains the outcome of a receipt verification.
type VerifyResult struct {
	Valid        bool              `json:"valid"`
	Checks       map[string]bool   `json:"checks"`
	FailureCodes []string          `json:"failure_codes"`
	RequestID    string            `json:"request_id"`
}

type bindResponse struct {
	Receipt   map[string]interface{} `json:"receipt"`
	RequestID string                 `json:"request_id"`
	ElapsedMs float64                `json:"elapsed_ms"`
}
