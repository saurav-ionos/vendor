package DataObjects

type Response struct {
	Result   string      `json:"result"`
	ErrorRes interface{} `json:"error"`
}

type LoginInfoObj struct {
	UserName   string `json:"user-email"`
	TenantName string `json:"tenant-name"`
	Password   string `json:"password"`
}

type LoginInputObj struct {
	Input LoginInfoObj `json:"input"`
}

type CpeInfoObj struct {
	SessionId string `json:"sessionid"`
}

type CpeInputObj struct {
	Input CpeInfoObj `json:"input"`
}

type CpeDetailObj struct {
	SessionId string `json:"sessionid"`
	CpeId     string `json:"cpeId"`
}
type CpeDetailObjInputObj struct {
	Input CpeDetailObj `json:"input"`
}

type CpeLogLevelInfo struct {
	SessionId string `json:"sessionid"`
	CpeId     string `json:"cpeId"`
	LogLevel  string `json:"logLevel"`
}
