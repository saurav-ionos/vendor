package Handler

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"syscall"

	logr "github.com/Sirupsen/logrus"
	Const "github.com/ionosnetworks/qfx_dp/cpewizard/Constants"
	DBManager "github.com/ionosnetworks/qfx_dp/cpewizard/DataBase"
	DataObj "github.com/ionosnetworks/qfx_dp/cpewizard/DataObjects"
	FileManager "github.com/ionosnetworks/qfx_dp/cpewizard/FileManager"
	User "github.com/ionosnetworks/qfx_dp/cpewizard/User"
	Utility "github.com/ionosnetworks/qfx_dp/cpewizard/Utilities"
	Sqlite3Util "github.com/ionosnetworks/qfx_dp/cpewizard/sqlite3Util"
)

func CheckLogin(w http.ResponseWriter, r *http.Request) {
	logr.Info("Processing Login")
	response := make(map[string]Response)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	var usrInfo DataObj.UserInfoObj
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))

	if err != nil {
		logr.Error("Error reading request body", err.Error())
		response["output"] = Response{Result: "FAILURE", ErrorRes: "INVALID_REQUEST"}
	} else if err := json.Unmarshal(body, &usrInfo); err != nil {
		response["output"] = Response{Result: "FAILURE", ErrorRes: Const.InvalidData}
	} else {
		//usersData := DBManager.GetUserInfo()
		usrJson, err := User.GetUsers()
		if err != nil && DBManager.CreateUserInfoObj() && usrInfo.UserName == "debug@ionos.com" && Utility.GetHashValueOfString(usrInfo.Password) == Utility.GetHashValueOfString("ionos") {
			authToken := Utility.SecureRandomAlphaNumericString(32)
			DBManager.UpdateUserInfoWithToken(authToken, usrInfo)
			response["output"] = Response{Result: "SUCCESS", CPInfo: DBManager.GetCPInfo(), PortConfigs: []DataObj.PortConfigInfo{}, AuthToken: authToken, ErrorRes: Const.NoError}
			go DBManager.ClearAuthTokenAfterTime(Const.TokenDur, authToken)
		} else if err != nil && !DBManager.CreateUserInfoObj() {
			response["output"] = Response{Result: "FAILURE", ErrorRes: "INTERNAL_ERROR"}
		} else {
			//DBManager.CreateUserInfoObj()
			DBManager.UpdateDbUsers()
			userExists := false
			for _, user := range usrJson.Users {
				if user.UserName == usrInfo.UserName && user.Password == Utility.GetHashValueOfString(usrInfo.Password) {
					userExists = true
				}
			}
			if userExists {
				authToken := Utility.SecureRandomAlphaNumericString(32)
				DBManager.UpdateUserInfoWithToken(authToken, usrInfo)
				response["output"] = Response{Result: "SUCCESS", CPInfo: DBManager.GetCPInfo(), PortConfigs: []DataObj.PortConfigInfo{}, AuthToken: authToken, ErrorRes: Const.NoError}
				go DBManager.ClearAuthTokenAfterTime(Const.TokenDur, authToken)
			} else {
				response["output"] = Response{Result: "FAILURE", ErrorRes: Const.InvalidUser}
			}
		}
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

func GetCpeId(w http.ResponseWriter, r *http.Request) {
	cpeId := FileManager.ReadCpeId()
	response := make(map[string]CpeIdResponse)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	response["output"] = CpeIdResponse{Result: "SUCCESS", CpeId: cpeId, ErrorRes: Const.NoError}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}
func GetCPDetails(w http.ResponseWriter, r *http.Request) {
	logr.Debug("Getting CP Details")
	authtoken := r.Header.Get("X-Auth-Token")
	//existingAuthTok, tokenErr := DBManager.GetCurrentAuthToken()
	response := make(map[string]Response)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	if DBManager.ValidateAuthToken(authtoken) {
		response["output"] = Response{Result: "SUCCESS", CPInfo: DBManager.GetCPInfo(), PortConfigs: nil, AuthToken: "", ErrorRes: Const.NoError}
	} else {
		response["output"] = Response{Result: "FAILURE", ErrorRes: Const.InvalidToken}
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

func GetPortInfoDetails(w http.ResponseWriter, r *http.Request) {
	logr.Debug("Getting Port Info")
	authtoken := r.Header.Get("X-Auth-Token")
	//existingAuthTok, tokenErr := DBManager.GetCurrentAuthToken()
	response := make(map[string]Response)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	if !DBManager.ValidateAuthToken(authtoken) {
		response["output"] = Response{Result: "FAILURE", ErrorRes: Const.InvalidToken}
	} else if len(Const.NonEditablePhysicalEth) <= 1 || len(Const.NonEditablePhysicalEth) <= 1 || len(Const.EthernetMapping) == 0 {
		response["output"] = Response{Result: "SUCCESS", CPInfo: DBManager.GetCPInfo(), PortConfigs: []DataObj.PortConfigInfo{}, AuthToken: "", ErrorRes: Const.NoError}
	} else {
		response["output"] = Response{Result: "SUCCESS", CPInfo: DBManager.GetCPInfo(), PortConfigs: FileManager.GetInterfaces(), AuthToken: "", ErrorRes: Const.NoError}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

func UpdatePortsInfoWithInfo(w http.ResponseWriter, r *http.Request) {
	logr.Debug("Updating Ports Info")
	var newPortsInfo []DataObj.PortConfigInfo
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))

	if err != nil {
		logr.Warning("Error decoding request body", err.Error())
	}
	authtoken := r.Header.Get("X-Auth-Token")
	//existingAuthTok, tokenErr := DBManager.GetCurrentAuthToken()
	response := make(map[string]Response)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	if err := json.Unmarshal(body, &newPortsInfo); err != nil {
		response["output"] = Response{Result: "FAILURE", ErrorRes: Const.InvalidData}
	} else if !DBManager.ValidateAuthToken(authtoken) {
		response["output"] = Response{Result: "FAILURE", ErrorRes: Const.InvalidToken}
	} else if validity, err := Utility.ValidateAllInterfaces(newPortsInfo, FileManager.GetInterfaces()); !validity && err != nil {
		response["output"] = Response{Result: "FAILURE", ErrorRes: err.Error()}
	} else if validity, err := FileManager.CheckMultiLink(newPortsInfo); !validity && err != nil {
		response["output"] = Response{Result: "FAILURE", ErrorRes: err.Error()}
	} else {
		FileManager.CreateInterfaceBkpFolder(Const.InterfaceBkpFolder)
		FileManager.CopyInterfaceFiles(Const.InterfaceBkpFolder)
		FileManager.UpdateInterfaceFilesNew(newPortsInfo)
		erro := DBManager.UpdatePortsInfoWithNewInfo(newPortsInfo)
		if erro != nil {
			logr.Warning("Error updating database with updated interfaces", erro.Error())
		}
		response["output"] = Response{Result: "SUCCESS", ErrorRes: Const.NoError}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

func GetWanStatusForInterfaces(w http.ResponseWriter, r *http.Request) {
	logr.Info("Getting Wan Status")
	authtoken := r.Header.Get("X-Auth-Token")
	response := make(map[string]WanResponse)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	if !DBManager.ValidateAuthToken(authtoken) {
		response["output"] = WanResponse{Result: "FAILURE", ErrorRes: Const.InvalidToken} //Response{Result: "SUCCESS", CPInfo: DBManager.GetCPInfo(), PortConfigs: DBManager.GetPortConfigs(), AuthToken: "", ErrorRes: NoError}
	} else {
		response["output"] = WanResponse{Result: "SUCCESS", WanStatus: Utility.GetWanStatusForInterfaces(DBManager.GetCPInfo()), ErrorRes: Const.NoError} //Response{Result: "SUCCESS", CPInfo: DBManager.GetCPInfo(), PortConfigs: DBManager.GetPortConfigs(), AuthToken: "", ErrorRes: NoError}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

func GetConnectivity(w http.ResponseWriter, r *http.Request) {
	logr.Info("Getting General Connectivity")
	FileManager.CreateCmdLogFile()
	FileManager.CreateLogTarFile(Const.LogFileLocation)
	authtoken := r.Header.Get("X-Auth-Token")
	response := make(map[string]INetResponse)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	if !DBManager.ValidateAuthToken(authtoken) {
		response["output"] = INetResponse{Result: "FAILURE", ErrorRes: Const.InvalidToken} //Response{Result: "SUCCESS", CPInfo: DBManager.GetCPInfo(), PortConfigs: DBManager.GetPortConfigs(), AuthToken: "", ErrorRes: NoError}
	} else {
		var status bool
		var err string
		if Utility.CheckInternetConnectivityWithPing(Const.PingCheckIP) && Utility.DnsResolution(Const.DnsResolvHost) {
			status = true
			err = Const.NoError
		} else if !Utility.CheckInternetConnectivityWithPing(Const.PingCheckIP) {
			status = false
			err = Const.InetConnFail
		} else if !Utility.DnsResolution(Const.DnsResolvHost) {
			status = false
			err = Const.DnsResolvFail
		}
		response["output"] = INetResponse{Result: "SUCCESS", Status: status, ErrorRes: err, LogPath: Const.CurrentLogSoftLink}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

func GetControllerConnectivity(w http.ResponseWriter, r *http.Request) {
	logr.Info("Getting Controller Connectivity")
	FileManager.CreateCmdLogFile()
	FileManager.CreateLogTarFile(Const.LogFileLocation)
	authtoken := r.Header.Get("X-Auth-Token")
	response := make(map[string]INetResponse)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	if !DBManager.ValidateAuthToken(authtoken) {
		response["output"] = INetResponse{Result: "FAILURE", ErrorRes: Const.InvalidToken} //Response{Result: "SUCCESS", CPInfo: DBManager.GetCPInfo(), PortConfigs: DBManager.GetPortConfigs(), AuthToken: "", ErrorRes: NoError}
	} else {
		response["output"] = INetResponse{Result: "SUCCESS", Status: Utility.GetIc2Connectivity(), ErrorRes: Const.NoError, LogPath: Const.CurrentLogSoftLink}
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

func GetLoggerFile(w http.ResponseWriter, r *http.Request) {
}

func Logout(w http.ResponseWriter, r *http.Request) {
	logr.Info("Logging out")
	authtoken := r.Header.Get("X-Auth-Token")
	response := make(map[string]Response)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	DBManager.ClearAuthToken(authtoken)
	DBManager.InvalidateTokenTimer(authtoken)
	response["output"] = Response{Result: "SUCCESS", ErrorRes: Const.NoError}
	w.WriteHeader(StatusSuccess)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

func GetRootDirTree(w http.ResponseWriter, r *http.Request) {
	logr.Info("GetRootDirTree")

	fileList := []DataObj.File{}
	response := make(map[string]DirDataResponse)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	var dirInfoObj DataObj.DirInfoObj
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logr.Error("Error reading request body", err.Error())
		response["output"] = DirDataResponse{Result: "FAILURE", FileList: []DataObj.File{}, ErrorRes: "INVALID_REQUEST"}
	} else if err := json.Unmarshal(body, &dirInfoObj); err != nil {
		response["output"] = DirDataResponse{Result: "FAILURE", FileList: []DataObj.File{}, ErrorRes: Const.InvalidData}
	} else {
		logr.Info("Need to get Dir Info of path: ", Const.CpeRootDir)
		pathId := Sqlite3Util.GetPathId(Const.CpeRootDir + "/")
		dirFileCount := Sqlite3Util.GetTotalDirFileCount(pathId)
		fileList = Sqlite3Util.GetRootDirs(pathId)
		response["output"] = DirDataResponse{Result: "SUCCESS", FileList: fileList,
			DirDetailInfo: DataObj.DirDetailInfo{RootPath: Const.CpeRootDir, FreeSpace: DiskFreeSpace(Const.CpeRootDir), TotalFileCount: dirFileCount}, ErrorRes: Const.NoError}
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

// disk usage of path/disk
func DiskFreeSpace(path string) uint64 {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return 0
	}
	return fs.Bfree * uint64(fs.Bsize)
}

func GetDirInfo(w http.ResponseWriter, r *http.Request) {
	logr.Info("GetCpeDirInfo")
	response := make(map[string]DirDataResponse)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	var dirInfoObj DataObj.DirInfoObj
	var dirFileCount int
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logr.Error("Error reading request body", err.Error())
		response["output"] = DirDataResponse{Result: "FAILURE", FileList: []DataObj.File{}, ErrorRes: "INVALID_REQUEST"}
	} else if err := json.Unmarshal(body, &dirInfoObj); err != nil {
		response["output"] = DirDataResponse{Result: "FAILURE", FileList: []DataObj.File{}, ErrorRes: Const.InvalidData}
	} else {
		logr.Info("Need to get Dir Info of given request***: ", dirInfoObj)
		pathId := Sqlite3Util.GetPathId(dirInfoObj.DirFullPath)
		fileList := Sqlite3Util.GetDirInfo(pathId, dirInfoObj.DirFullPath, dirInfoObj.PageNum, dirInfoObj.SearchPattern)
		if dirInfoObj.SearchPattern != "" {
			dirFileCount = Sqlite3Util.GetTotalSearchCount(dirInfoObj.SearchPattern, dirInfoObj.DirFullPath)
		} else {
			dirFileCount = Sqlite3Util.GetTotalDirFileCount(pathId)
		}
		response["output"] = DirDataResponse{Result: "SUCCESS", FileList: fileList,
			DirDetailInfo: DataObj.DirDetailInfo{RootPath: Const.CpeRootDir, FreeSpace: DiskFreeSpace(Const.CpeRootDir), TotalFileCount: dirFileCount}, ErrorRes: Const.NoError}
	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

func ReloadDir(w http.ResponseWriter, r *http.Request) {
	logr.Info("ReloadDir")

	fileList := []DataObj.File{}
	response := make(map[string]DirDataResponse)
	w.Header().Set("Content-Type", "application/json;charset=UTF-8")
	w.WriteHeader(StatusSuccess)
	var dirInfoObj DataObj.DirInfoObj
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		logr.Error("Error reading request body", err.Error())
		response["output"] = DirDataResponse{Result: "FAILURE", FileList: []DataObj.File{}, ErrorRes: "INVALID_REQUEST"}
	} else if err := json.Unmarshal(body, &dirInfoObj); err != nil {
		response["output"] = DirDataResponse{Result: "FAILURE", FileList: []DataObj.File{}, ErrorRes: Const.InvalidData}
	} else {
		logr.Info("Need to Reload Dir for path!!!: ", dirInfoObj.DirFullPath)
		Sqlite3Util.ClearExistingData(dirInfoObj.DirFullPath)
		logr.Info("Deleted Dir for path: ", dirInfoObj.DirFullPath)

		Sqlite3Util.WalkAndUpdateDB(dirInfoObj.DirFullPath)
		logr.Info("Updated Dir for path: ", dirInfoObj.DirFullPath)

		response["output"] = DirDataResponse{Result: "SUCCESS", FileList: fileList,
			DirDetailInfo: DataObj.DirDetailInfo{RootPath: Const.CpeRootDir, FreeSpace: DiskFreeSpace(Const.CpeRootDir), TotalFileCount: 0}, ErrorRes: Const.NoError}

	}
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logr.Error("Error encoding json", response, err.Error())
	}
}

//func checkDnsResolution() error {
//	interfaces := DBManager.GetPortConfigs()
//	for _, iface := range interfaces {
//		if len(iface.PrimaryDns) > 0 && !Utility.DnsResolution(iface.PrimaryDns) {
//			return errors.New(Const.PrimaryDnsFail + "_" + Const.EthernetMapping[iface.PrimaryLan])
//		} else if len(iface.SecondaryDns) > 0 && !Utility.DnsResolution(iface.SecondaryDns) {
//			return errors.New(Const.SecondaryDnsFail + "_" + Const.EthernetMapping[iface.PrimaryLan])
//		}
//	}
//	return nil
//}
