package crypto

import (
	"testing"
)

func TestFieldEncryptor(t *testing.T) {
	aesKey := []byte("01234567890123456789012345678901") // 32 bytes
	hmacKey := []byte("hmac_secret_key_that_is_long_enough_32bytes")

	tests := []struct {
		name      string
		aes       []byte
		hmac      []byte
		plaintext string
		wantErr   bool
	}{
		{
			name:      "Success Encrypt Decrypt",
			aes:       aesKey,
			hmac:      hmacKey,
			plaintext: "my sensitive data",
			wantErr:   false,
		},
		{
			name:      "Empty string",
			aes:       aesKey,
			hmac:      hmacKey,
			plaintext: "",
			wantErr:   false,
		},
		{
			name:      "Invalid AES key length",
			aes:       []byte("short"),
			hmac:      hmacKey,
			plaintext: "data",
			wantErr:   true, // NewFieldEncryptor should fail
		},
		{
			name:      "Invalid HMAC key length",
			aes:       aesKey,
			hmac:      []byte("short"),
			plaintext: "data",
			wantErr:   true, // NewFieldEncryptor should fail
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			enc, err := NewFieldEncryptor(tt.aes, tt.hmac)
			if err != nil {
				if !tt.wantErr {
					t.Fatalf("unexpected error creating encryptor: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("expected error, got nil")
			}

			cipher, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			plain, err := enc.Decrypt(cipher)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			if plain != tt.plaintext {
				t.Errorf("got %q, want %q", plain, tt.plaintext)
			}
		})
	}
}

func TestBlindIndex(t *testing.T) {
	aesKey := []byte("01234567890123456789012345678901") // 32 bytes
	hmacKey := []byte("hmac_secret_key_that_is_long_enough_32bytes")

	enc, err := NewFieldEncryptor(aesKey, hmacKey)
	if err != nil {
		t.Fatalf("unexpected error creating encryptor: %v", err)
	}

	index1 := enc.BlindIndex("data")
	index2 := enc.BlindIndex("data")
	index3 := enc.BlindIndex("other")

	if index1 == "" {
		t.Error("expected blind index to not be empty")
	}

	if index1 != index2 {
		t.Error("blind index should be deterministic")
	}

	if index1 == index3 {
		t.Error("blind index should be different for different inputs")
	}

    // test empty
    emptyIndex := enc.BlindIndex("")
    if emptyIndex != "" {
        t.Error("blind index for empty string should be empty")
    }
}

func TestEncryptedTypes(t *testing.T) {
	aesKey := []byte("01234567890123456789012345678901") // 32 bytes
	hmacKey := []byte("hmac_secret_key_that_is_long_enough_32bytes")

	enc, err := NewFieldEncryptor(aesKey, hmacKey)
	if err != nil {
		t.Fatalf("unexpected error creating encryptor: %v", err)
	}

	SetGlobalEncryptor(enc)
	defer SetGlobalEncryptor(nil)

	// EncryptedString
	es := EncryptedString("test")
	val, err := es.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}
	var scannedES EncryptedString
	err = scannedES.Scan(val)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if scannedES != es {
		t.Errorf("got %s, want %s", scannedES, es)
	}

	// EncryptedFloat
	ef := EncryptedFloat(123.45)
	valF, err := ef.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}
	var scannedEF EncryptedFloat
	err = scannedEF.Scan(valF)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if scannedEF != ef {
		t.Errorf("got %f, want %f", scannedEF, ef)
	}
}

func TestEncryptWithIndex(t *testing.T) {
	aesKey := []byte("01234567890123456789012345678901") // 32 bytes
	hmacKey := []byte("hmac_secret_key_that_is_long_enough_32bytes")

	enc, err := NewFieldEncryptor(aesKey, hmacKey)
	if err != nil {
		t.Fatalf("unexpected error creating encryptor: %v", err)
	}

	encStr, indexStr, err := enc.EncryptWithIndex("my data")
	if err != nil {
		t.Fatalf("EncryptWithIndex error: %v", err)
	}
	if encStr == "" || indexStr == "" {
		t.Errorf("got empty string")
	}

	// test empty
	encEmpty, indexEmpty, err := enc.EncryptWithIndex("")
	if err != nil {
		t.Fatalf("EncryptWithIndex error: %v", err)
	}
	if encEmpty != "" || indexEmpty != "" {
		t.Errorf("expected empty string")
	}
}

func TestEncryptedTypesNoGlobal(t *testing.T) {
	SetGlobalEncryptor(nil)

	// EncryptedString
	es := EncryptedString("test")
	val, err := es.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}
	if val.(string) != "test" {
		t.Errorf("expected test, got %v", val)
	}

	var scannedES EncryptedString
	err = scannedES.Scan(nil)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if scannedES != "" {
		t.Errorf("expected empty")
	}

	err = scannedES.Scan([]byte("test"))
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if scannedES != "test" {
		t.Errorf("expected test")
	}

	// error scan
	err = scannedES.Scan(123)
	if err == nil {
		t.Errorf("expected error scanning int")
	}

	// EncryptedFloat
	ef := EncryptedFloat(123.45)
	valF, err := ef.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}
	if valF.(string) != "123.45" {
		t.Errorf("expected 123.45, got %v", valF)
	}

	var scannedEF EncryptedFloat
	err = scannedEF.Scan(nil)
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if scannedEF != 0 {
		t.Errorf("expected 0")
	}

	err = scannedEF.Scan([]byte("123.45"))
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	if scannedEF != 123.45 {
		t.Errorf("expected 123.45")
	}

	err = scannedEF.Scan(123)
	if err == nil {
		t.Errorf("expected error scanning int")
	}
}
