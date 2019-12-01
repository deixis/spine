package crypto_test

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"testing"

	"github.com/pkg/errors"
	"github.com/deixis/spine/crypto"
)

const (
	keySize = 32
)

var (
	testMessage = []byte("It can be whatever, but nðt meant to last forever.")
	testKey     []byte
)

func TestRotor(t *testing.T) {
	// Create keys
	keys := map[uint32][]byte{}
	for _, id := range []uint32{42, 43, 45} {
		key, err := genRandBytes(keySize)
		if err != nil {
			t.Fatalf("%v", err)
		}
		keys[id] = key
	}
	rotor := crypto.NewRotor(keys, 43)

	// Default key
	ct, err := rotor.Encrypt(testMessage)
	if err != nil {
		t.Fatalf("%v", err)
	}
	out, err := rotor.Decrypt(ct)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !bytes.Equal(out, testMessage) {
		t.Fatal("messages don't match")
	}

	// Explicit key
	ct, err = rotor.EncryptWithSender(testMessage, 42)
	if err != nil {
		t.Fatalf("%v", err)
	}
	out, err = rotor.Decrypt(ct)
	if err != nil {
		t.Fatalf("%v", err)
	}
	if !bytes.Equal(out, testMessage) {
		t.Fatal("messages don't match")
	}

	// Invalid key
	newSender := make([]byte, 4)
	binary.BigEndian.PutUint32(newSender, 49)
	for i := 0; i < 4; i++ {
		ct[i] = newSender[i]
	}
	_, err = rotor.Decrypt(ct)
	if err == nil {
		t.Fatal("decryption should fail with invalid AD")
	}
}

func genRandBytes(l int) ([]byte, error) {
	b := make([]byte, l)
	if _, err := rand.Read(b); err != nil {
		return nil, errors.Wrap(err, "rand error")
	}
	return b, nil
}
