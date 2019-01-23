package random

import "crypto/rand"

var LatinAndNumbers = []byte("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")

func Bytes(count int) []byte {
	res := make([]byte, count)
	_, err := rand.Read(res)
	if err != nil {
		panic(err)
	}
	return res
}

func String(count int, chars []byte) string {
	res := make([]byte, count)
	lenc := byte(len(chars))
	for i, b := range Bytes(count) {
		res[i] = chars[b%lenc]
	}
	return string(res)
}
