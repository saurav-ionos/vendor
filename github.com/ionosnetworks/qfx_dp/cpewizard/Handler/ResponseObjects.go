package Handler

import (
	DataObj "github.com/ionosnetworks/qfx_dp/cpewizard/DataObjects"
)

const (
	StatusSuccess = 200
)

type Response struct {
	//userInfo    DataObj.UserInfoObj      `json:userinfo`
	Result      string                   `json:"result"`
	CPInfo      DataObj.CPInfoObj        `json:"cpeInfo"`
	PortConfigs []DataObj.PortConfigInfo `json:"ethConfigs"`
	AuthToken   string                   `json:"token"`
	ErrorRes    interface{}              `json:"error"`
}

type WanResponse struct {
	Result    string      `json:"result"`
	WanStatus interface{} `json:"wanStatus"`
	ErrorRes  string      `json:"error"`
}

type INetResponse struct {
	Result   string `json:"result"`
	Status   bool   `json:"status"`
	LogPath  string `json:"logpath"`
	ErrorRes string `json:"error"`
}

type CpeIdResponse struct {
	Result   string `json:"result"`
	CpeId    string `json:"cpeId"`
	ErrorRes string `json:"error"`
}

type DirDataResponse struct {
	Result        string                `json:"result"`
	FileList      []DataObj.File        `json:"fileList"`
	DirDetailInfo DataObj.DirDetailInfo `json:dirDetailInfo`
	ErrorRes      string                `json:"error"`
}
