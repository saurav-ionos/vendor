package fakemsg

import (
	"fmt"
	"io"
	"net"
)

var cToAMap map[string]string
var handler func(io.Reader) error

type mcast struct {
	sink []io.Writer // collection of all the io.Writers to which the data is to be sent
}

func init() {
	cToAMap = make(map[string]string)
	cToAMap["ded731565cef841830b3160d068cbb55"] = "172.20.116.2"
	cToAMap["devregvinciv1bcpe200000000000000"] = "172.20.104.2"

	laddr := "0.0.0.0:12000"
	l, err := net.Listen("tcp4", laddr)
	if err != nil {
		panic(err)
	}
	go func() {
		for {
			c, e := l.Accept()
			fmt.Println("Accepted connection")
			if e == nil {
				go handler(c)
			}
		}
	}()
}

func UpdateHandler(t func(io.Reader) error) {
	handler = t
}

func NewMsgHandle(destCpeID string) io.ReadWriter {
	address, ok := cToAMap[destCpeID]
	if ok == false {
		return nil
	}

	raddr := address + ":12000"
	conn, err := net.Dial("tcp4", raddr)
	if err != nil {
		panic(err)
	}
	fmt.Println("connected to ", raddr)
	return conn
}

func NewMulticastHandler(destCpeIDs []string) (io.Writer, error) {

	var m *mcast
	if len(destCpeIDs) > 0 {
		m = new(mcast)
		for _, x := range destCpeIDs {
			address, ok := cToAMap[x]
			if ok == false {
				fmt.Println("No address entry for", x)
				continue
			}

			raddr := address + ":12000"
			conn, err := net.Dial("tcp4", raddr)
			if err != nil {
				panic(err)
			}
			fmt.Println("connected to ", raddr)
			m.sink = append(m.sink, conn)
		}
	}
	return m, nil
}

func (m *mcast) Write(b []byte) (int, error) {
loopSinks:
	for _, x := range m.sink {
		written := 0
		toWrite := len(b)
		for toWrite > 0 {
			n, e := x.Write(b[written:])
			if e != nil {
				fmt.Println("Could not write to", x, e)
				continue loopSinks
			}
			written += n
			toWrite -= n
		}
	}
	return len(b), nil
}
