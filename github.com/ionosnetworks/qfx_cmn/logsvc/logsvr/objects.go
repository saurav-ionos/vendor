package main

type LogChan struct {
	channel chan LogMsgRequest
}

func (lc LogChan) Write(msg []byte) (n int, e error) {
	lc.channel <- LogMsgRequest{Msg: string(msg[:])}
	return len(msg), nil
}

func (lc LogChan) Close() error {
	return nil
}

type LogMsgRequest struct {
	Msg string
}

type LogSvr struct {
	port            string
	Iparray         []string
	EtcdIP          string
	LogPipeLineIP   string
	LogPipeLinePort string
	maxWorkers      int
	maxLogQueueSize int
	LogMsgQueue     chan LogMsgRequest
	LogMsgCritQueue chan LogMsgRequest
	LogPipeLineType string
}
