//go:generate stringer -type=ApiReturnStatus
package objects

type ApiReturnStatus int

const (
	SUCCESS ApiReturnStatus = iota
	FAILURE ApiReturnStatus = iota
)

type ApiResp struct {
	Status    string `json:"status"`
	ErrorCode string `json:"errorcode"`
	ErrorDesc string `json:"errordesc"`
}

type SyncReln struct {
	Name       string   `json:"name"`
	SyncId     string   `json:"syncid"`
	TenantName string   `json:"tenantname"`
	UserName   string   `json:"clientname"`
	SrcCpeId   string   `json:"srccpeid"`
	DstCpeIds  []string `json:"dstcpeidlist"`
}

type CreateSyncRelnInput struct {
	SyncReln
}

type CreateSyncRelnOutput struct {
	ApiResp
}

type EditSyncRelnInput struct {
	SyncReln
}

type EditSyncRelnOutput struct {
	ApiResp
}

type GetSyncRelnInput struct {
	SyncId string `json:"syncid"`
	TenantName string   `json:"tenantname"`
}

type GetSyncRelnOutput struct {
	SyncReln
	ApiResp
}
