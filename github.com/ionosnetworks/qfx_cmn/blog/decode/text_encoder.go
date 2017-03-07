package decode

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/ionosnetworks/qfx_cmn/blog"
)

//Text encoder : Implements the blog.Encoder interface and generates
// a human readable text stream of logs

type textEncoder struct {
	w io.WriteCloser
}

func NewTextEncoder(w io.WriteCloser) *textEncoder {
	t := new(textEncoder)
	t.w = w
	return t
}

func (t *textEncoder) Encode(v interface{}) error {

	data := v.(blog.Fields)
	var w bytes.Buffer
	fmt.Fprintf(&w, "%s %s ctx=%s msg=%s", data["time"],
		strings.ToUpper(data["level"].(string)), data["ctx"], data["msg"])
	for key, val := range data {
		if key != "time" && key != "level" &&
			key != "ctx" && key != "msg" {
			fmt.Fprintf(&w, " %s=%v", key, val)
		}

	}
	fmt.Fprintf(&w, "\n")
	toWrite := len(w.Bytes())
	written := 0
	buf := w.Bytes()
	for toWrite > 0 {
		n, e := t.w.Write(buf[written:])
		if e != nil {
			return e
		}
		toWrite -= n
		written += n
	}
	return nil
}
