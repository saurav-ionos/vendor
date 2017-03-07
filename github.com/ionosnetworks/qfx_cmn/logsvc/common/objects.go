// All the object defentions for Log Server, Client and API
package common

// Message types
const (
	AUTH_MSG    = 1
	AUTH_MSG_OK = 2
)

type MsgHdr struct {
	Version  int32
	Magic    int32
	MetaSize int32
	MsgType  int32
}
