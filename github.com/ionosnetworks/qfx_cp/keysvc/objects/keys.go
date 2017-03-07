package objects

type KeyIn struct {
	AccessKey string `json:"accesskey"`
}

type KeyCredential struct {
	ApiList     []string `json:"apilist"`
	FeatureList []string `json:"featurelist"`
}

// Optional. if this is passed, we will inherit all the
// credentials of this key.
type KeyCreateIn struct {
	Name         string        `json:"name"`
	TenantName   string        `json:"tenantname"`
	ClientName   string        `json:"clientname"`
	AccessKey    string        `json:"accesskey"`
	AccessSecret string        `json:"accesssecret"`
	Credential   KeyCredential `json:"keycredential"`
}

type KeyModifyIn struct {
	AccessKey string `json:"accesskey"`

	Credential KeyCredential `json:"keycredential"`
	// Set, Add, Delete, or Inherit
	Operation string `json:"operation"`
}

type ClientKeyIn struct {
	Name       string `json:"name"`
	TenantName string `json:"tenantname"`
	ClientName string `json:"clientname"`
}

type KeyEntryOut struct {
	Name         string `json:"name"`
	TenantName   string `json:"tenantname"`
	ClientName   string `json:"clientname"`
	AccessKey    string `json:"accesskey"`
	AccessSecret string `json:"accesssecret"`

	Credential KeyCredential `json:"keycredential"`
	ErrCode    string        `json:"errcode"`
}

type KeyOut KeyEntryOut
type KeyEntryOutList struct {
	KeyIdList []string `json:"keyidlist"`
	ErrCode   string   `json:"errcode"`
}

type ObjectDeleteOut struct {
	ErrCode string `json:"errcode"`
}
