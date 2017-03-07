package objects

type ApiIn struct {
	AccessPoint string `json:"accesspoint"`
	Uri         string `json:"uri"`
}

type ApiOut struct {
	AccessPoint string `json:"accesspoint"`
	Uri         string `json:"uri"`
	ErrCode     string `json:"errcode"`
}

type ApiKey struct {
	AccessPoint string `json:"accesspoint"`
}

type ObjectDeleteOut struct {
	ErrCode string `json:"errcode"`
}
