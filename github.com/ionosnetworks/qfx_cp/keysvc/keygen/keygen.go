package main

import (
	"os"

	ka "github.com/ionosnetworks/qfx_cmn/keyreader"
	u "github.com/ionosnetworks/qfx_cp/keysvc/utils"
	cnts "github.com/ionosnetworks/qfx_cp/qfxConsts"
)

func main() {

	filename := "keyfile"
	key := ka.AccessKey{Version: 1, Magic: 11259375,
		Key:    u.SecureRandomAlphaNumeringString(cnts.ACCESS_KEY_LENGTH),
		Secret: u.SecureRandomAlphaNumeringString(cnts.ACCESS_SECRET_LENGTH)}

	if len(os.Args) >= 2 {
		filename = os.Args[1]
	}
	key.WriteToFile(filename)
}
