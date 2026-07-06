package logs_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"

	"bank-api/pkg/logs"
)

func TestBlockedFields_Exists(t *testing.T) {
	// Test that BlockedFields variable is defined and accessible
	if logs.BlockedFields == nil {
		t.Error("BlockedFields should not be nil")
	}
}

func TestBlockedFields_NotEmpty(t *testing.T) {
	// Test that BlockedFields contains at least some entries
	if len(logs.BlockedFields) == 0 {
		t.Error("BlockedFields should not be empty")
	}
}

func TestBlockedFields_ContainsCommonSecrets(t *testing.T) {
	// Test that common sensitive field names are included
	expectedFields := []string{
		"password",
		"token",
		"secret",
		"api_key",
		"credit_card",
		"ssn",
	}

	for _, expected := range expectedFields {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain %q", expected)
		}
	}
}

func TestBlockedFields_PasswordVariants(t *testing.T) {
	// Test that various password-related fields are blocked
	passwordVariants := []string{
		"password",
		"password_hash",
		"pass",
		"pwd",
	}

	for _, variant := range passwordVariants {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == variant {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain password variant %q", variant)
		}
	}
}

func TestBlockedFields_TokenVariants(t *testing.T) {
	// Test that various token-related fields are blocked
	tokenVariants := []string{
		"token",
		"access_token",
		"refresh_token",
	}

	for _, variant := range tokenVariants {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == variant {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain token variant %q", variant)
		}
	}
}

func TestBlockedFields_APIKeyVariants(t *testing.T) {
	// Test that various API key-related fields are blocked
	apiKeyVariants := []string{
		"api_key",
		"apikey",
	}

	for _, variant := range apiKeyVariants {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == variant {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain API key variant %q", variant)
		}
	}
}

func TestBlockedFields_AuthenticationFields(t *testing.T) {
	// Test that authentication-related fields are blocked
	authFields := []string{
		"authorization",
		"auth",
		"bearer",
	}

	for _, field := range authFields {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain authentication field %q", field)
		}
	}
}

func TestBlockedFields_CookieFields(t *testing.T) {
	// Test that cookie-related fields are blocked
	cookieFields := []string{
		"cookie",
		"set-cookie",
	}

	for _, field := range cookieFields {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain cookie field %q", field)
		}
	}
}

func TestBlockedFields_PII(t *testing.T) {
	// Test that PII (Personally Identifiable Information) fields are blocked
	piiFields := []string{
		"ssn",
		"email",
		"phone",
		"address",
	}

	for _, field := range piiFields {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain PII field %q", field)
		}
	}
}

func TestBlockedFields_FinancialData(t *testing.T) {
	// Test that financial data fields are blocked
	financialFields := []string{
		"credit_card",
		"card_number",
		"cvv",
		"account_number",
		"iban",
	}

	for _, field := range financialFields {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain financial field %q", field)
		}
	}
}

func TestBlockedFields_CryptographicKeys(t *testing.T) {
	// Test that cryptographic key fields are blocked
	keyFields := []string{
		"secret_key",
		"private_key",
	}

	for _, field := range keyFields {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain cryptographic key field %q", field)
		}
	}
}

func TestBlockedFields_MFAFields(t *testing.T) {
	// Test that MFA/2FA related fields are blocked
	mfaFields := []string{
		"otp",
		"mfa",
		"totp",
	}

	for _, field := range mfaFields {
		found := false
		for _, blocked := range logs.BlockedFields {
			if blocked == field {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("BlockedFields should contain MFA field %q", field)
		}
	}
}

func TestBlockedFields_NoDuplicates(t *testing.T) {
	// Test that there are no duplicate entries in BlockedFields
	seen := make(map[string]bool)
	for _, field := range logs.BlockedFields {
		if seen[field] {
			t.Errorf("BlockedFields contains duplicate entry: %q", field)
		}
		seen[field] = true
	}
}

func TestBlockedFields_AllLowercase(t *testing.T) {
	// Test that all blocked fields are lowercase (for consistent matching)
	for _, field := range logs.BlockedFields {
		// Check if field contains uppercase characters
		for _, char := range field {
			if char >= 'A' && char <= 'Z' {
				t.Errorf("BlockedFields entry %q contains uppercase characters, should be lowercase", field)
				break
			}
		}
	}
}

func TestBlockedFields_Count(t *testing.T) {
	// Test that we have a reasonable number of blocked fields
	// This is a sanity check to ensure the list is comprehensive
	minExpectedFields := 10
	if len(logs.BlockedFields) < minExpectedFields {
		t.Errorf("BlockedFields has %d entries, expected at least %d", len(logs.BlockedFields), minExpectedFields)
	}
}

func TestBlockedFields_Integration(t *testing.T) {
	// Integration test: verify BlockedFields works with SanitizingHandler
	var buf bytes.Buffer
	base := slog.NewJSONHandler(&buf, nil)
	handler := logs.NewSanitizingHandler(base, logs.BlockedFields)
	logger := slog.New(handler)

	logger.Info("test",
		"username", "john",
		"password", "secret123",
		"email", "john@example.com",
		"token", "abc-token",
		"api_key", "my-api-key",
		"safe_field", "safe-value",
	)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse log JSON: %v", err)
	}

	if result["password"] != "****" {
		t.Errorf("password should be masked, got %v", result["password"])
	}
	if result["token"] != "****" {
		t.Errorf("token should be masked, got %v", result["token"])
	}
	if result["email"] != "****" {
		t.Errorf("email should be masked, got %v", result["email"])
	}
	if result["safe_field"] != "safe-value" {
		t.Errorf("safe_field should not be masked, got %v", result["safe_field"])
	}
}

func TestBlockedFields_CoverageCheck(t *testing.T) {
	// Verify that all major categories of sensitive data are covered
	categories := map[string][]string{
		"passwords":   {"password", "pwd"},
		"tokens":      {"token"},
		"secrets":     {"secret"},
		"api_keys":    {"api_key", "apikey"},
		"auth":        {"authorization", "auth"},
		"cookies":     {"cookie"},
		"pii":         {"ssn", "email"},
		"financial":   {"credit_card", "cvv"},
		"crypto_keys": {"private_key"},
		"mfa":         {"otp"},
	}

	for category, fields := range categories {
		for _, field := range fields {
			found := false
			for _, blocked := range logs.BlockedFields {
				if blocked == field {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Category %q missing field %q in BlockedFields", category, field)
			}
		}
	}
}

func TestBlockedFields_SpecificEntries(t *testing.T) {
	// Test for specific entries that should definitely be in the list
	tests := []struct {
		name  string
		field string
	}{
		{"has password", "password"},
		{"has password_hash", "password_hash"},
		{"has token", "token"},
		{"has access_token", "access_token"},
		{"has refresh_token", "refresh_token"},
		{"has id_token", "id_token"},
		{"has api_key", "api_key"},
		{"has secret", "secret"},
		{"has secret_key", "secret_key"},
		{"has authorization", "authorization"},
		{"has bearer", "bearer"},
		{"has cookie", "cookie"},
		{"has ssn", "ssn"},
		{"has credit_card", "credit_card"},
		{"has cvv", "cvv"},
		{"has private_key", "private_key"},
		{"has otp", "otp"},
		{"has email", "email"},
		{"has phone", "phone"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := false
			for _, blocked := range logs.BlockedFields {
				if blocked == tt.field {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("BlockedFields should contain %q", tt.field)
			}
		})
	}
}
