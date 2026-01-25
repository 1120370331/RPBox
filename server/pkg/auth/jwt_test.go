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
