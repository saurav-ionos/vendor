package auth

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	flatbuffers "github.com/google/flatbuffers/go"
	"github.com/ionosnetworks/qfx_dp/fb"
)

type MsgHdr struct {
	Version uint32
	Magic   uint32
	Length  uint32
	MsgID   uint32
}

const (
	MsgIDAuthMesg  uint32 = 1
	MsgIDAuthReply uint32 = 2
)

func Authenticate(where io.ReadWriter, accessKey string, secretKey string) error {

	//build the auth message
	b := flatbuffers.NewBuilder(0)

	p := b.CreateByteString([]byte("nodeid"))
	q := b.CreateByteString([]byte("nodestr"))
	s := b.CreateByteString([]byte("secret"))
	t := b.CreateByteString([]byte("accesskey"))
	fb.AuthmesgStart(b)
	fb.AuthmesgAddNodeID(b, p)
	fb.AuthmesgAddNodeName(b, q)
	fb.AuthmesgAddSecret(b, s)
	fb.AuthmesgAddAccessKey(b, t)

	r := fb.AuthmesgEnd(b)

	b.Finish(r)

	buf := b.Bytes[b.Head():]

	// create the header
	hdr := MsgHdr{Version: 1, Magic: 0xaabb,
		Length: uint32(len(buf)), MsgID: MsgIDAuthMesg}

	err := binary.Write(where, binary.LittleEndian, &hdr)
	if err != nil {
		fmt.Println("header writing failed")
		return err
	}
	// Send the message to the server
	toWrite := len(buf)
	written := 0
	for toWrite > 0 {
		n, e := where.Write(buf[written:])
		if e != nil {
			return err
		}
		written += n
		toWrite -= n
	}

	// Wait for auth ok message
	buf = make([]byte, 16)
	err = binary.Read(where, binary.LittleEndian, &hdr)
	if err != nil {
		fmt.Println("error reading data", err)
		return err
	}
	if hdr.MsgID == MsgIDAuthReply {
		return nil
	}
	return errors.New("Auth declined")
}
