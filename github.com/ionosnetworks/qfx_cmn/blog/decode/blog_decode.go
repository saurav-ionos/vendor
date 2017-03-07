package decode

import (
	"encoding/gob"
	"errors"
	"io"

	"github.com/ionosnetworks/qfx_cmn/blog"
)

type Xcoder struct {
	d Decoder            // The source to read from
	e map[string]Encoder // The format to convert to
	r io.ReadCloser      // The input stream
	//out io.WriteCloser      Where to write the the new encoded stream
}

type Encoder interface {
	Encode(v interface{}) error
}

type Decoder interface {
	Decode(v interface{}) error
}

func NewXcoder(r io.Reader, e map[string]Encoder) *Xcoder {
	x := new(Xcoder)
	x.d = gob.NewDecoder(r)
	x.e = e
	//x.out = out
	return x
}

func (x *Xcoder) Xcode() error {
	var err error = nil
	v := new(blog.Entry)
	err = x.d.Decode(v)
	if err == nil {
		isEncoder := false
		v.Data["time"] = v.T.UTC().Format("Mon Jan 2 15:04:05.00000 UTC 2006")
		v.Data["level"] = v.L.String()
		if v.Count > 1 {
			v.Data["count"] = v.Count
		}
		for lev, enc := range x.e {
			if lev == v.Data["level"] {
				isEncoder = true
				err = enc.Encode(v.Data)
			}
		}
		if !isEncoder {
			err = errors.New("Encoder not found for log level in the map")
		}
	}
	return err
}

func (x *Xcoder) Close() {
	x.r.Close()
	//x.out.Close()
}
