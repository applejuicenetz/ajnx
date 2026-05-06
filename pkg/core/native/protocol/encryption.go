package protocol

import (
	"crypto/rand"
	"io"
)

// Encrypter kapselt die appleJuice XOR-Verschlüsselung für einen Stream.
type Encrypter struct {
	key   [8]byte
	index int
}

// NewEncrypter erstellt einen neuen Encrypter mit zufälligen Keys.
func NewEncrypter() (*Encrypter, error) {
	e := &Encrypter{}
	_, err := rand.Read(e.key[:])
	return e, err
}

// NewDecrypter erstellt einen neuen Decrypter (Encrypter) mit einem gegebenen Key.
func NewDecrypter(key [8]byte) *Encrypter {
	return &Encrypter{key: key}
}

// Key gibt den aktuellen 8-Byte Key zurück.
func (e *Encrypter) Key() []byte {
	return e.key[:]
}

// Encrypt verschlüsselt ein einzelnes Byte (ausgehend).
func (e *Encrypter) Encrypt(b byte) byte {
	e.index++
	keyByte := e.key[e.index%8]
	// result = ((input ^ key) - index) % 256
	res := (int(b ^ keyByte) - e.index) % 256
	if res < 0 {
		res += 256
	}
	return byte(res)
}

// Decrypt entschlüsselt ein einzelnes Byte (eingehend).
func (e *Encrypter) Decrypt(b byte) byte {
	e.index++
	keyByte := e.key[e.index%8]
	// result = ((input + index) % 256) ^ key
	res := (int(b) + e.index) % 256
	return byte(res) ^ keyByte
}

// EncryptBuffer verschlüsselt einen ganzen Buffer in-place.
func (e *Encrypter) EncryptBuffer(buf []byte) {
	for i := range buf {
		buf[i] = e.Encrypt(buf[i])
	}
}

// DecryptBuffer entschlüsselt einen ganzen Buffer in-place.
func (e *Encrypter) DecryptBuffer(buf []byte) {
	for i := range buf {
		buf[i] = e.Decrypt(buf[i])
	}
}

// ReadEncrypted liest verschlüsselte Daten und entschlüsselt sie.
func (e *Encrypter) ReadEncrypted(r io.Reader, buf []byte) (int, error) {
	n, err := io.ReadFull(r, buf)
	if n > 0 {
		e.DecryptBuffer(buf[:n])
	}
	return n, err
}

// WriteEncrypted verschlüsselt Daten und schreibt sie.
func (e *Encrypter) WriteEncrypted(w io.Writer, buf []byte) (int, error) {
	// Wir kopieren den Buffer, um das Original nicht zu verändern
	encrypted := make([]byte, len(buf))
	copy(encrypted, buf)
	e.EncryptBuffer(encrypted)
	return w.Write(encrypted)
}
