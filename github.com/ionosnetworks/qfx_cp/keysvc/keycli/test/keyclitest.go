package main

import (
	"fmt"
	"os"

	kcli "github.com/ionosnetworks/qfx_cp/keysvc/keycli"
)

func main() {

	key := os.Args[1]
	secret := os.Args[2]

	if cli, err := kcli.New(key, secret); cli != nil {

		if key, err := cli.Get(os.Args[3]); err == nil {
			fmt.Println("Key : ", key)
		} else {
			fmt.Println("failed to get key ")
		}
	} else {
		fmt.Println("Failed to initialize ", err)
	}
}
