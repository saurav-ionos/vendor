package infra

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

type CsID [16]byte

func (c CsID) String() string {
	var s string
	s = fmt.Sprintf("%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x%02x",
		c[0], c[1], c[2], c[3], c[4], c[5],
		c[6], c[7], c[8], c[9], c[10], c[11],
		c[12], c[13], c[14], c[15])
	return s
}

type UUID [16]byte

func (b UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

var csid CsID

const csIDpath string = "/etc/ionos-cpeid.conf"

func init() {
	fi, err := os.Open(csIDpath)
	if err != nil {
		return
		//		panic(err)
	}
	localCsID, err := bufio.NewReader(fi).ReadString('\n')
	if err != io.EOF {
		panic(err)
	}
	b := []byte(localCsID)
	dst := make([]byte, hex.DecodedLen(len(b)))
	_, err = hex.Decode(dst, b)
	if err != nil {
		panic(err)
	}
	copy(csid[:], dst)
}

func GetLocalCsID() CsID {
	return csid
}

func DecodeStringToCsID(s string) CsID {
	var c CsID
	b := []byte(s)
	d := make([]byte, hex.DecodedLen(len(b)))
	_, _ = hex.Decode(d, b)
	copy(c[:], d)
	return c
}

func GenUUID() UUID {
	f, _ := os.Open("/dev/urandom")
	var b UUID
	f.Read(b[:])
	f.Close()
	return b
}
