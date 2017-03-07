package keyreader

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

type AccessKey struct {
	Version int    `json:"version"`
	Magic   int    `json:"magic"`
	Key     string `json:"key"`
	Secret  string `json:"secret"`
}

func (key *AccessKey) Encode(w io.Writer) {

	if err := json.NewEncoder(w).Encode(key); err != nil {
		fmt.Println("Failed to encode", err)
	}
}

func (key *AccessKey) Decode(r io.Reader) {
	if err := json.NewDecoder(r).Decode(key); err != nil {
		fmt.Println("Failed to Decode", err)
	}
}

func (key *AccessKey) WriteToFile(filename string) error {

	if byteArr, err := json.Marshal(key); err == nil {

		if err = ioutil.WriteFile(filename, byteArr, os.ModeExclusive); err != nil {
			fmt.Println("File error: %v\n", err)
			return err
		}
	} else {
		return err
	}
	return nil
}

func New(filename string) AccessKey {
	var key AccessKey

	if file, e := ioutil.ReadFile(filename); e != nil {
		fmt.Println("File error: %v\n", e)
		return key
	} else {
		json.Unmarshal(file, &key)
	}

	return key
}
