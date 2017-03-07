package Utility

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"

	logr "github.com/Sirupsen/logrus"
)

const (
	alphaBytes    = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphaNumeric  = "1234567890" + alphaBytes
	letterIdxBits = 6
	letterIdxMask = 1<<letterIdxBits - 1
)

// ConvertObjectToByteArray convert object to byte array
func ConvertObjectToByteArray(object interface{}) ([]byte, error) {
	var myDataBuff bytes.Buffer
	newEncoder := gob.NewEncoder(&myDataBuff)
	encodeErr := newEncoder.Encode(object)

	if encodeErr != nil {
		logr.Error("Error encoding object", object, encodeErr.Error())
		return nil, encodeErr
	}

	return myDataBuff.Bytes(), nil
}

//ConvertByteArrayToObject convert byte array to generic interface
func ConvertByteArrayToObject(byteArray []byte, object interface{}) {
	newDecoder := gob.NewDecoder(bytes.NewBuffer(byteArray))
	decodeErr := newDecoder.Decode(object)
	if decodeErr != nil {
		logr.Error("error decoding object", object, decodeErr.Error())
	}
}

func RemoveEmptyStringsFromArray(dataArray []string) []string {
	var newDataArray []string
	for _, currentElement := range dataArray {
		if !(len(currentElement) == 0 || currentElement == " " || currentElement == "\n") {
			newDataArray = append(newDataArray, currentElement)
		}
	}
	return newDataArray
}

func SecureRandomAlphaString(length int) string {
	return SecureRandomString(alphaBytes, length)
}

func SecureRandomAlphaNumericString(length int) string {
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

func SecureRandomBytes(length int) []byte {
	var randomBytes = make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		logr.Error("Unable to generate random bytes")
	}
	return randomBytes
}

func GetHashValueOfString(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	//fmt.Println(hex.EncodeToString(h.Sum(nil)))
	//fmt.Printf("%x", h.Sum(nil))
	return hex.EncodeToString(h.Sum(nil))

}
