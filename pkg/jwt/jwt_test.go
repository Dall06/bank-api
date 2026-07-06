package jwt

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerator(t *testing.T) {
	config := Config{
		Secret:     "super-secret",
		Expiration: time.Hour,
	}
	g := NewGenerator(config)

	// Test Generate
	input := GenerateInput{
		UserID:    "user123",
		StaffID:   "staff456",
		CompanyID: "comp789",
		Email:     "test@test.com",
		Role:      "admin",
	}

	out, err := g.Generate(input)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}
	if out.Token == "" {
		t.Errorf("expected token, got empty string")
	}

	// Test GenerateTempToken
	tempOut, err := g.GenerateTempToken("user123", "test@test.com", 5*time.Minute)
	if err != nil {
		t.Fatalf("GenerateTempToken failed: %v", err)
	}
	if tempOut.Token == "" {
		t.Errorf("expected temp token, got empty string")
	}

	// Verify temp token
	tempClaims, err := g.ValidateTempToken(tempOut.Token)
	if err != nil {
		t.Fatalf("ValidateTempToken failed: %v", err)
	}
	if tempClaims.UserID != "user123" {
		t.Errorf("got user ID %s, want user123", tempClaims.UserID)
	}
	if tempClaims.Type != "company_selection" {
		t.Errorf("got type %s, want company_selection", tempClaims.Type)
	}

	// Verify normal token
	claims, err := g.Validate(out.Token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if claims.UserID != "user123" {
		t.Errorf("got user ID %s, want user123", claims.UserID)
	}
	if claims.Role != "admin" {
		t.Errorf("got role %s, want admin", claims.Role)
	}

	// Test invalid tokens
	_, err = g.Validate("invalid")
	if err == nil {
		t.Errorf("expected error parsing invalid token")
	}
	_, err = g.ValidateTempToken("invalid")
	if err == nil {
		t.Errorf("expected error parsing invalid temp token")
	}
}

func TestValidateWithWrongSecret(t *testing.T) {
	// Setup
	config := Config{Secret: "secret", Expiration: time.Hour}
	g := NewGenerator(config)

	out, _ := g.Generate(GenerateInput{UserID: "u1", Email: "test@t.com"})

	// Validate with a different generator that has a wrong secret
	gWrong := NewGenerator(Config{Secret: "wrong", Expiration: time.Hour})

	_, err := gWrong.Validate(out.Token)
	if err == nil {
		t.Error("expected error validating with wrong secret")
	}
}

func TestParseTokenWithCustomClaims(t *testing.T) {
    // Test parsing a token signed with wrong alg or missing claims
    token := jwt.New(jwt.SigningMethodNone)
    tokenString, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
    
	g := NewGenerator(Config{Secret: "secret", Expiration: time.Hour})
    _, err := g.Validate(tokenString)
    if err == nil {
        t.Error("expected error for token with wrong signature method")
    }
}
