package core

import (
	cmnCpeo "github.com/ionosnetworks/qfx_cmn/qfxObjects"
	cpeo "github.com/ionosnetworks/qfx_cp/cpemgr/objects"
)

var (
	cpeList = []cmnCpeo.CpeInfo{

		{Name: "cpeTest Name1", CpeId: "CPE1000DJKH88877", PhysicalCpeId: "VCPE10001", HwVersion: "v2",
			ModelNumber: "1234", SerialNumber: "6789", Tenant: "tenant1", User: "user1", Region: "Region1",
			UploadBW: 0, DownloadBW: 0, AllocatedBW: 0, FwVersion: "fw1.0.1", Status: 0},

		{Name: "cpeTest Name2", CpeId: "CPE1001KDSJ88899", PhysicalCpeId: "VCPE10002", HwVersion: "v2",
			ModelNumber: "12345", SerialNumber: "6790", Tenant: "tenant2", User: "user2", Region: "Region2",
			UploadBW: 0, DownloadBW: 0, AllocatedBW: 0, FwVersion: "fw1.0.1", Status: 0},
		{Name: "cpeTest Name2", CpeId: "CPE1002DSD66655", PhysicalCpeId: "VCPE10002", HwVersion: "v2",
			ModelNumber: "12345", SerialNumber: "6790", Tenant: "tenant2", User: "user2", Region: "Region2",
			UploadBW: 0, DownloadBW: 0, AllocatedBW: 0, FwVersion: "fw1.0.1", Status: 0},
	}

	providerList = []string{"AMAZON", "GOOGLE"}

	awzsiteList = []string{"Americas", "EMEA", "Asia"}
	gwsiteList  = []string{"Northeastern Asia-Pacific", "Eastern Asia-Pacific ", "Western Europe",
		"Eastern US", "Central US", "Western US"}

	zoneList = []string{"Northern Virginia", "Ohio", "Oregon", "Northern California",
		"Montreal", "São Paulo", "GovCloud"}
)

func GetAllCpe() cpeo.CpeResponse {
	return cpeo.CpeResponse{Result: "SUCCESS", Cpes: cpeList}
}

func AddCpe(cpe cmnCpeo.CpeInfo) cpeo.CpeResponse {
	cpeList = append(cpeList, cpe)
	return cpeo.CpeResponse{Result: "SUCCESS", Cpes: cpeList}
}

func GetAllProvider() cpeo.ProviderResponse {
	return cpeo.ProviderResponse{Result: "SUCCESS", Providers: providerList}
}

func GetAllSite(provider string) cpeo.SiteResponse {
	var list = []string{}
	switch provider {
	case "AMAZON":
		return cpeo.SiteResponse{Result: "SUCCESS", Sites: awzsiteList}
	case "GOOGLE":
		return cpeo.SiteResponse{Result: "SUCCESS", Sites: gwsiteList}
	}
	return cpeo.SiteResponse{Result: "SUCCESS", Sites: list}
}

func GetAllZone(site string) cpeo.ZoneResponse {
	return cpeo.ZoneResponse{Result: "SUCCESS", Zones: zoneList}
}

func AddRegion(region cmnCpeo.Region) cpeo.RegionResponse {
	var list = []string{}
	return cpeo.CpeResponse{Result: "SUCCESS", Regions: list}
}
