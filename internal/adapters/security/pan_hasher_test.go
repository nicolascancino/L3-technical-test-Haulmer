package security

import "testing"

func TestHMACPANHasher_Hash(t *testing.T) {
	hasher := NewHMACPANHasher("salt")

	first, err := hasher.Hash("4111111111111111")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	second, _ := hasher.Hash("4111111111111111")
	if first != second {
		t.Errorf("same pan and salt gave %q and %q", first, second)
	}
	if len(first) != 64 {
		t.Errorf("hash length = %d, want 64 hex characters", len(first))
	}

	otherPAN, _ := hasher.Hash("5500000000000004")
	if otherPAN == first {
		t.Error("different pan gave the same hash")
	}
	otherSalt, _ := NewHMACPANHasher("other").Hash("4111111111111111")
	if otherSalt == first {
		t.Error("different salt gave the same hash")
	}
}
