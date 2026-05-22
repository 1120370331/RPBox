package auth

import "testing"

func TestGenerateAndParseToken(t *testing.T) {
	Init("test-secret")

	token, err := GenerateToken(42, "tester", 1)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	if claims.UserID != 42 {
		t.Fatalf("expected user id 42, got %d", claims.UserID)
	}
	if claims.Username != "tester" {
		t.Fatalf("expected username tester, got %s", claims.Username)
	}
}

func TestParseTokenWithWrongSecret(t *testing.T) {
	Init("secret-a")
	token, err := GenerateToken(1, "tester", 1)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	Init("secret-b")
	if _, err := ParseToken(token); err == nil {
		t.Fatalf("expected parse error with wrong secret")
	}
}

func TestGenerateAndParseSwitchToken(t *testing.T) {
	Init("test-secret")

	token, expiresAt, err := GenerateSwitchToken(7, "switcher", 60)
	if err != nil {
		t.Fatalf("generate switch token: %v", err)
	}
	if expiresAt.IsZero() {
		t.Fatalf("expected switch token expiry")
	}

	claims, err := ParseSwitchToken(token)
	if err != nil {
		t.Fatalf("parse switch token: %v", err)
	}
	if claims.UserID != 7 {
		t.Fatalf("expected user id 7, got %d", claims.UserID)
	}
	if claims.Username != "switcher" {
		t.Fatalf("expected username switcher, got %s", claims.Username)
	}
}
