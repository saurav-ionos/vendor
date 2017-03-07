package utils

import (
	"crypto/rand"
	"log"
) //import

const (
	alphaBytes    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ" // 52 possibilities
	alphaNumeric  = "1234567890" + alphaBytes                              // 52 possibilities
	letterIdxBits = 6                                                      // 6 bits to represent 64 possibilities / indexes
	letterIdxMask = 1<<letterIdxBits - 1                                   // All 1-bits, as many as letterIdxBits
)

func SecureRandomAlphaString(length int) string {

	return SecureRandomString(alphaBytes, length)
}

func SecureRandomAlphaNumeringString(length int) string {

	return SecureRandomString(alphaNumeric, length)
}

func SecureRandomString(charset string, length int) string {

	result := make([]byte, length)
	bufferSize := int(float64(length) * 1.3)
	for i, j, randomBytes := 0, 0, []byte{}; i < length; j++ {
		if j%bufferSize == 0 {
			randomBytes = SecureRandomBytes(bufferSize)
		}
		if idx := int(randomBytes[j%length] & letterIdxMask); idx < len(charset) {
			result[i] = charset[idx]
			i++
		}
	}

	return string(result)
}

// SecureRandomBytes returns the requested number of bytes using crypto/rand
func SecureRandomBytes(length int) []byte {
	var randomBytes = make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		log.Fatal("Unable to generate random bytes")
	}
	return randomBytes
}

/*
func main() {

	fmt.Println("  32 bytes ", SecureRandomAlphaString(32))
	fmt.Println("  32 bytes ", SecureRandomAlphaString(32))
}
*/
