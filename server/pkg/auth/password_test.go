package auth

import "testing"

func TestHashPassword(t *testing.T) {
	password := "s3cret-pass"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == password {
		t.Fatalf("hash should not match plaintext")
	}
	if !CheckPassword(password, hash) {
		t.Fatalf("expected password to validate")
	}
	if CheckPassword("wrong", hash) {
		t.Fatalf("expected password mismatch")
	}
}
