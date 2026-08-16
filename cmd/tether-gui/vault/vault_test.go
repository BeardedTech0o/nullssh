package vault

import "testing"

func TestEncryptDecryptRoundTrip(t *testing.T) {
	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	key, err := DeriveKey("correct horse battery staple", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	ciphertext, err := EncryptString(key, "hunter2")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	if ciphertext == "hunter2" {
		t.Fatal("EncryptString returned the plaintext unmodified")
	}

	plaintext, err := DecryptString(key, ciphertext)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if plaintext != "hunter2" {
		t.Fatalf("DecryptString() = %q, want %q", plaintext, "hunter2")
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}
	rightKey, err := DeriveKey("right password", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	wrongKey, err := DeriveKey("wrong password", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	ciphertext, err := EncryptString(rightKey, "secret")
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}

	if _, err := DecryptString(wrongKey, ciphertext); err == nil {
		t.Fatal("DecryptString with the wrong key succeeded, want error")
	}
}

func TestDeriveKeyIsDeterministic(t *testing.T) {
	salt, err := NewSalt()
	if err != nil {
		t.Fatalf("NewSalt: %v", err)
	}

	k1, err := DeriveKey("same password", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}
	k2, err := DeriveKey("same password", salt)
	if err != nil {
		t.Fatalf("DeriveKey: %v", err)
	}

	if string(k1) != string(k2) {
		t.Fatal("DeriveKey produced different keys for the same password and salt")
	}
}
