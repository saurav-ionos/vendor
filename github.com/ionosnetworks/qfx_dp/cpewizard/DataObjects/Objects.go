package DataObjects

import (
	"time"
)

type UserInfoObj struct {
	UserName  string `json:"userName"`
	Password  string `json:"passWord"`
	AuthToken string `json:"authToken"`
}

type CPInfoObj struct {
	CPName          string `json:"name"`
	Location        string `json:"location"`
	CPeId           string `json:"cpeId"`
	CPeStatus       string `json:"cpeStatus"`
	CPeVersion      string `json:"cpVersion"`
	TimeZone        string `json:"timeZone"`
	CloudSeedUptime string `json:"uptime"`
	NetworkStatus   string `json:"networkStatus"`
	Storage         string `json:"storage"`
}

type PortConfigInfo struct {
	PrimaryLan      string `json:"primaryLan"`
	PhysicalPort    string `json:"physicalPort"`
	IPAddressType   string `json:"ipType"`
	IPAddress       string `json:"ipAddress"`
	Subnet          string `json:"subnet"`
	GateWay         string `json:"gateway"`
	LanSpeed        string `json:"lanSpeed"`
	IsEditable      bool   `json:"isEditable"`
	LanStatus       bool   `json:"lanStatus"`
	LinkStatus      bool   `json:"linkStatus"`
	PrimaryDns      string `json:"primaryDns"`
	SecondaryDns    string `json:"secondaryDns"`
	MacAddress      string `json:"macAddress"`
	MultiLinkActive bool   `json:"multiLinkActive"`
}

type WanStatusInfoV1 struct {
	Eth0 string `json:"eth0"`
	Eth1 string `json:"eth1"`
	Eth2 string `json:"eth2"`
	Eth3 string `json:"eth3"`
}

type WanStatusInfoV2 struct {
	Eth0 string `json:"eth0"`
	Eth1 string `json:"eth1"`
	Eth2 string `json:"eth2"`
	Eth3 string `json:"eth3"`
	Eth4 string `json:"eth4"`
	Eth5 string `json:"eth5"`
}

type ErrorObj struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
}

type LoggerParams struct {
	LogLevel string `json:"level"`
	LogFile  string `json:"file"`
}

type ConfigDetails struct {
	LogParams LoggerParams `json:"logparams"`
}

type ConfigObj struct {
	Config ConfigDetails `json:"config"`
}

type DirInfoObj struct {
	DirFullPath   string `json:"dirFullPath"`
	PageNum       int    `json:"pageNum"`
	SearchPattern string `json:"searchPattern"`
}

type DirDataObj struct {
	ContentName  string `json:"contentName"`
	IsDir        bool   `json:"isDir"`
	TotalSize    uint64 `json:"totalSize"`
	ModifiedTime int    `json:"modifiedTime"`
	IsExported   bool   `json:"isExported"`
}

type File struct {
	PathId     int       `json:PathId`
	FilePath   string    `json:"filePath"`
	FileName   string    `json:"fileName"`
	FileSize   int64     `json:"fileSize"`
	IsDir      bool      `json:"isDir"`
	IsExported bool      `json:"isExported"`
	Level      int       `json:"level"`
	ModTime    time.Time `json:"modTime"`
}

type DirDetailInfo struct {
	RootPath       string `json:"rootPath"`
	FreeSpace      uint64 `json:"freeSpace"`
	TotalFileCount int    `json:"totalFileCount"`
}
