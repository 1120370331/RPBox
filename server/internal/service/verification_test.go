package service

import (
	"context"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestVerificationServiceCodeFlow(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	svc := NewVerificationService(client)

	code, err := svc.GenerateCode()
	if err != nil {
		t.Fatalf("generate code: %v", err)
	}
	if len(code) != 6 {
		t.Fatalf("expected 6-digit code, got %s", code)
	}

	if err := svc.SaveCode(ctx, "user@example.com", code); err != nil {
		t.Fatalf("save code: %v", err)
	}

	ok, err := svc.VerifyCode(ctx, "user@example.com", code)
	if err != nil {
		t.Fatalf("verify code: %v", err)
	}
	if !ok {
		t.Fatalf("expected code to verify")
	}

	ok, err = svc.VerifyCode(ctx, "user@example.com", code)
	if err != nil {
		t.Fatalf("verify code again: %v", err)
	}
	if ok {
		t.Fatalf("expected code to be cleared after verification")
	}
}

func TestVerificationRateLimit(t *testing.T) {
	ctx := context.Background()
	mr := miniredis.RunT(t)

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	svc := NewVerificationService(client)

	ok, err := svc.CheckRateLimit(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("check rate limit: %v", err)
	}
	if !ok {
		t.Fatalf("expected rate limit to allow first request")
	}

	ok, err = svc.CheckRateLimit(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("check rate limit again: %v", err)
	}
	if ok {
		t.Fatalf("expected rate limit to block second request")
	}
}
