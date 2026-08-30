package key

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const Prefix = "smtp"

func Generate() (plain, hash string, err error) {
	buf := make([]byte, 32)

	if _, err = rand.Read(buf); err != nil {
		return
	}
	plain = Prefix + base64.RawURLEncoding.EncodeToString(buf)
	return plain, Hash(plain), nil
}

func Hash(source string) string {
	sum := sha256.Sum256([]byte(source))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
