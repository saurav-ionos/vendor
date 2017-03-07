package objects

import (
	cpeo "github.com/ionosnetworks/qfx_cmn/qfxObjects"
)

type CpeResponse struct {
	Result string         `json:"result"`
	Cpes   []cpeo.CpeInfo `json:"cpes"`
}

type ProviderResponse struct {
	Result    string   `json:"result"`
	Providers []string `json:"providers"`
}

type SiteResponse struct {
	Result string   `json:"result"`
	Sites  []string `json:"sites"`
}

type ZoneResponse struct {
	Result string   `json:"result"`
	Zones  []string `json:"zones"`
}
