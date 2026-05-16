package hashutil

import (
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestHashAndVerify(t *testing.T) {
	hash, err := Hash("my-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if err := Verify("my-password", hash); err != nil {
		t.Errorf("verify: %v", err)
	}
}

func TestVerify_WrongPassword(t *testing.T) {
	hash, _ := Hash("correct")
	if Verify("wrong", hash) == nil {
		t.Error("expected error for wrong password")
	}
}

func TestHash_MinCost(t *testing.T) {
	hash, _ := Hash("pw")
	c, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		t.Fatalf("cost parse: %v", err)
	}
	if c < 12 {
		t.Errorf("want cost >= 12, got %d", c)
	}
}

func TestHash_UniqueOutputs(t *testing.T) {
	// Same plaintext produces different hashes (bcrypt salt)
	h1, _ := Hash("same-password")
	h2, _ := Hash("same-password")
	if h1 == h2 {
		t.Error("bcrypt must produce unique hashes for the same input")
	}
}
