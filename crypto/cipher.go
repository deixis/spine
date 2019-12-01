package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"

	"github.com/pkg/errors"
)

const (
	// CKeySize is the cipher key size - AES-256
	CKeySize = 32
	// MKeySize is the HMAC key size - HMAC-SHA-256
	MKeySize = 32
	// KeySize is the encryption key size
	KeySize = CKeySize + MKeySize
	// AESNonceSize is an AES nonce size
	AESNonceSize = aes.BlockSize
	// GCMNonceSize is a GCM nonce size
	GCMNonceSize = 12
	// SenderSize is the size allocated to add the sender ID
	SenderSize = 4
	// MACSize MAC size
	MACSize = 32
)

var (
	// ErrEncrypt occurs when the encryption process fails. The reason of failure
	// is concealed for security reason
	ErrEncrypt = errors.New("sec: encryption failed")
	// ErrDecrypt occurs when the decryption process fails.
	ErrDecrypt = errors.New("sec: decryption failed")
)

// Encrypt secures a message using AES-GCM.
func Encrypt(key, message []byte) ([]byte, error) {
	c, err := aes.NewCipher(key[:CKeySize])
	if err != nil {
		return nil, ErrEncrypt
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, ErrEncrypt
	}

	nonce, err := genRandBytes(GCMNonceSize)
	if err != nil {
		return nil, ErrEncrypt
	}

	// Seal will append the output to the first argument; the usage
	// here appends the ciphertext to the nonce. The final parameter
	// is any additional data to be authenticated.
	out := gcm.Seal(nonce, nonce, message, nil)
	return out, nil
}

// Decrypt recovers a message secured using AES-GCM.
func Decrypt(key, message []byte) ([]byte, error) {
	if len(message) <= GCMNonceSize {
		return nil, ErrDecrypt
	}

	c, err := aes.NewCipher(key[:CKeySize])
	if err != nil {
		return nil, ErrDecrypt
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, ErrDecrypt
	}

	nonce := make([]byte, GCMNonceSize)
	copy(nonce, message)

	out, err := gcm.Open(nil, nonce, message[GCMNonceSize:], nil)
	if err != nil {
		return nil, ErrDecrypt
	}
	return out, nil
}

// Rotor is a encryption/decryption tool that supports key rotation
//
// Note: Data encrypted with sec.Encrypt cannot be decrypted with Rotor
type Rotor struct {
	keys          map[uint32][]byte
	defaultSender uint32

	NonceSize int
}

// NewRotor creates a new Rotor with the given keys.
// The defaultSender will be used as the default sender ID during the encryption process
func NewRotor(keys map[uint32][]byte, defaultSender uint32) *Rotor {
	return &Rotor{
		keys:          keys,
		defaultSender: defaultSender,
		NonceSize:     GCMNonceSize,
	}
}

// Encrypt secures a message and prepends the default 4-byte sender ID to the message.
func (r *Rotor) Encrypt(message []byte) ([]byte, error) {
	return r.EncryptWithSender(message, r.defaultSender)
}

// EncryptWithSender secures a message and prepends the given 4-byte sender ID to the message.
func (r *Rotor) EncryptWithSender(message []byte, sender uint32) ([]byte, error) {
	key, ok := r.keys[sender]
	if !ok {
		return nil, ErrEncrypt
	}

	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, sender)

	c, err := aes.NewCipher(key[:CKeySize])
	if err != nil {
		return nil, ErrEncrypt
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, ErrEncrypt
	}

	nonce, err := genRandBytes(r.NonceSize)
	if err != nil {
		return nil, ErrEncrypt
	}

	buf = append(buf, nonce...)
	buf = gcm.Seal(buf, nonce, message, buf[:4])
	return buf, nil
}

// Decrypt takes an incoming message and uses the sender ID to
// retrieve the appropriate key. It then attempts to recover the message
// using that key.
func (r *Rotor) Decrypt(message []byte) ([]byte, error) {
	if len(message) <= r.NonceSize+4 {
		return nil, ErrDecrypt
	}

	sender := binary.BigEndian.Uint32(message[:4])
	key, ok := r.keys[sender]
	if !ok {
		return nil, ErrDecrypt
	}

	c, err := aes.NewCipher(key[:CKeySize])
	if err != nil {
		return nil, ErrDecrypt
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, ErrDecrypt
	}

	nonce := make([]byte, r.NonceSize)
	copy(nonce, message[4:])

	// Decrypt the message, using the sender ID as the additional
	// data requiring authentication.
	out, err := gcm.Open(nil, nonce, message[4+r.NonceSize:], message[:4])
	if err != nil {
		return nil, ErrDecrypt
	}
	return out, nil
}

func genRandBytes(l int) ([]byte, error) {
	b := make([]byte, l)
	if _, err := rand.Read(b); err != nil {
		return nil, errors.Wrap(err, "rand error")
	}
	return b, nil
}
