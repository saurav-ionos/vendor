// Code generated by "stringer -type=ApiReturnStatus"; DO NOT EDIT

package objects

import "fmt"

const _ApiReturnStatus_name = "SUCCESSFAILURE"

var _ApiReturnStatus_index = [...]uint8{0, 7, 14}

func (i ApiReturnStatus) String() string {
	if i < 0 || i >= ApiReturnStatus(len(_ApiReturnStatus_index)-1) {
		return fmt.Sprintf("ApiReturnStatus(%d)", i)
	}
	return _ApiReturnStatus_name[_ApiReturnStatus_index[i]:_ApiReturnStatus_index[i+1]]
}
