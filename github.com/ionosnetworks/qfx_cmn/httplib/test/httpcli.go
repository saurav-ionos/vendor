package main

import (
	"fmt"
	"os"
	"strings"

	hcli "github.com/ionosnetworks/qfx_cmn/httplib/client"
)

var accessPoint = "192.168.56.101:9090"

func main() {

	operation := os.Args[1]
	command := os.Args[2]
	query := os.Args[3] // " { \"accesspoint\" : \"/path3\" } "

	cli := hcli.New(accessPoint)

	result, err := cli.RunCommand(operation, command, strings.NewReader(query))

	if err == nil {
		fmt.Println(string(result), err)
	} else {
		fmt.Println(err)
	}
}
