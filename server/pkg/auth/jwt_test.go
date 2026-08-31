package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

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

func TestParseTokenRejectsUnsafeTokens(t *testing.T) {
	Init("test-secret")
	now := time.Now()

	tests := []struct {
		name  string
		token func(t *testing.T) string
	}{
		{
			name: "malformed token",
			token: func(t *testing.T) string {
				return "not-a-jwt"
			},
		},
		{
			name: "none algorithm",
			token: func(t *testing.T) string {
				return signNoneToken(t, Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}})
			},
		},
		{
			name: "HS384 algorithm",
			token: func(t *testing.T) string {
				return signToken(t, jwt.SigningMethodHS384, Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}})
			},
		},
		{
			name: "missing expiration",
			token: func(t *testing.T) string {
				return signToken(t, jwt.SigningMethodHS256, Claims{UserID: 1})
			},
		},
		{
			name: "expired",
			token: func(t *testing.T) string {
				return signToken(t, jwt.SigningMethodHS256, Claims{UserID: 1, RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(now.Add(-time.Minute))}})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseToken(test.token(t)); err == nil {
				t.Fatal("ParseToken() error = nil, want rejection")
			}
		})
	}
}

func TestParseSwitchTokenRejectsUnsafeTokens(t *testing.T) {
	Init("test-secret")
	now := time.Now()
	validClaims := func(expiresAt *jwt.NumericDate) SwitchClaims {
		return SwitchClaims{
			UserID:  1,
			Purpose: accountSwitchPurpose,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: expiresAt,
			},
		}
	}

	tests := []struct {
		name  string
		token func(t *testing.T) string
	}{
		{
			name: "none algorithm",
			token: func(t *testing.T) string {
				return signNoneToken(t, validClaims(jwt.NewNumericDate(now.Add(time.Hour))))
			},
		},
		{
			name: "HS384 algorithm",
			token: func(t *testing.T) string {
				return signToken(t, jwt.SigningMethodHS384, validClaims(jwt.NewNumericDate(now.Add(time.Hour))))
			},
		},
		{
			name: "missing expiration",
			token: func(t *testing.T) string {
				return signToken(t, jwt.SigningMethodHS256, validClaims(nil))
			},
		},
		{
			name: "expired",
			token: func(t *testing.T) string {
				return signToken(t, jwt.SigningMethodHS256, validClaims(jwt.NewNumericDate(now.Add(-time.Minute))))
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := ParseSwitchToken(test.token(t)); err == nil {
				t.Fatal("ParseSwitchToken() error = nil, want rejection")
			}
		})
	}
}

func TestTokenPurposeIsolation(t *testing.T) {
	Init("test-secret")

	bearerToken, err := GenerateToken(1, "bearer", 1)
	if err != nil {
		t.Fatalf("GenerateToken(): %v", err)
	}
	if _, err := ParseSwitchToken(bearerToken); err == nil {
		t.Fatal("ParseSwitchToken() accepted a bearer token")
	}

	switchToken, _, err := GenerateSwitchToken(1, "switch", 1)
	if err != nil {
		t.Fatalf("GenerateSwitchToken(): %v", err)
	}
	if _, err := ParseToken(switchToken); err == nil {
		t.Fatal("ParseToken() accepted an account-switch token")
	}
}

func signToken(t *testing.T, method jwt.SigningMethod, claims jwt.Claims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString(jwtSecret)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func signNoneToken(t *testing.T, claims jwt.Claims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}
	return token
}
