package net

type ICipher interface {
	Init()
	Encrypt(seq int64, key uint32, data []byte) []byte
	Decrypt(seq int64, key uint32, data []byte) ([]byte, error)
}
