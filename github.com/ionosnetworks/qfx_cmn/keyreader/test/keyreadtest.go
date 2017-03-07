package main

import (
	"fmt"

	kr "github.com/ionosnetworks/qfx_cmn/keyreader"
)

var key kr.AccessKey = kr.AccessKey{Version: 1, Magic: 0xabcdef,
	Key: "testkey", Secret: "testSecret"}

func main() {

	writeKeyToFile()
	key2 := readKeyFromFile()

	if key != key2 {
		fmt.Println("FAIL:: Keys are different ")
	} else {
		fmt.Println("Pass:: ", key, key2)
	}
	encodeDecode(key2)
}

func writeKeyToFile() {
	key.WriteToFile("./keyfile")
}

func readKeyFromFile() kr.AccessKey {
	return kr.New("./keyfile")
}

func encodeDecode(key kr.AccessKey) {

}
