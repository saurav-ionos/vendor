package blog

import (
	"crypto/tls"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"sync"
)

type ntwrkblog struct {
	blogger
	param *serverParams
}

type serverParams struct {
	addr   string
	key    string
	secret string
	config *tls.Config
	conn   net.Conn
}

func New(logSrvAddr, accesskey, secret string) Logger {
	if conn, err := connect(logSrvAddr, accesskey, secret, 5); err != nil {
		fmt.Println("Failed to connect to log server ", err.Error())
		return nil
	} else {
		if err = authenticateSession(conn, accesskey, secret); err != nil {
			fmt.Println("Authorization failed")
			return nil
		}
		params := new(serverParams)
		params.addr = logSrvAddr
		params.key = accesskey
		params.secret = secret
		params.conn = conn
		return InitLogger(conn, params)

	}
}

func LazyLog(logSrvAddr, accesskey, secret string) Logger {
	params := new(serverParams)
	params.addr = logSrvAddr
	params.key = accesskey
	params.secret = secret
	ntwrkbl := new(ntwrkblog)
	bl := new(blogger)
	bl.outChan = make(chan *Entry, loggerMaxChanLen)
	bl.reconn = make(chan struct{})
	bl.L = Debug
	bl.entryPool.New = func() interface{} {
		return &Entry{}
	}
	bl.done = make(chan struct{})
	bl.Lock = &sync.Mutex{}
	ntwrkbl.blogger = *bl
	ntwrkbl.param = params
	go ntwrkbl.drainingLogs()
	go ntwrkbl.lazyconnect()
	return ntwrkbl
}
func (nbl *ntwrkblog) lazyconnect() {
	nbl.blogger.reconn <- struct{}{}
	nbl.reconnect()
	//if err != nil {
	//	fmt.Println("Error establishing connection", err.Error())
	//}
	nbl.blogger.reconn <- struct{}{}
}

func InitLogger(out io.WriteCloser, params *serverParams) Logger {
	ntwrkbl := new(ntwrkblog)
	bl := new(blogger)
	bl.outChan = make(chan *Entry, loggerMaxChanLen)
	bl.reconn = make(chan struct{})
	bl.L = Debug
	bl.entryPool.New = func() interface{} {
		return &Entry{}
	}
	bl.out = out
	bl.done = make(chan struct{})
	bl.Lock = &sync.Mutex{}
	bl.enc = gob.NewEncoder(out)
	ntwrkbl.blogger = *bl
	ntwrkbl.param = params
	go ntwrkbl.drainingLogs()
	go ntwrkbl.conncheck()
	return ntwrkbl
}

func (nbl *ntwrkblog) conncheck() {
	buf := make([]byte, 1024)
	for {
		_, err := nbl.param.conn.Read(buf)
		if err != nil {
			nbl.blogger.reconn <- struct{}{}
			fmt.Println("Error reading from conn", err.Error())
			if err.Error() == "use of closed network connection" {
				return
			} else if e, ok := err.(*net.OpError); ok {
				if e.Err.Error() == "use of closed network connection" {
					return
				}
			}
			nbl.blogger.enc = nil
			nbl.reconnect()
			//if err != nil {
			//	fmt.Println("Error reconnecting to server")
			//	return
			//}
			nbl.blogger.reconn <- struct{}{}
		}
	}
}

func (nbl *ntwrkblog) reconnect() {
	fmt.Println("reconnecting")
	if conn, err := connect(nbl.param.addr, nbl.param.key, nbl.param.secret, 5); err != nil {
		fmt.Println("Failed to reconnect to log server ", err.Error())
		nbl.reconnect()
		//return err
	} else {
		if err = authenticateSession(conn, nbl.param.key, nbl.param.secret); err != nil {
			fmt.Println("ConnectAuthorization failed")
			return
		}

		fmt.Println("connected again")
		nbl.param.conn = conn
		nbl.updateout(conn)
	}
	return
}

func (nbl *ntwrkblog) updateout(out io.WriteCloser) {
	nbl.blogger.Lock.Lock()
	fmt.Println("updating")
	nbl.blogger.out = out
	nbl.blogger.enc = gob.NewEncoder(out)
	nbl.blogger.Lock.Unlock()
}
