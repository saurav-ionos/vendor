package DataBase

import (
	"errors"
	"time"

	logr "github.com/Sirupsen/logrus"
	Const "github.com/ionosnetworks/qfx_dp/cpewizard/Constants"
	DataObj "github.com/ionosnetworks/qfx_dp/cpewizard/DataObjects"
	User "github.com/ionosnetworks/qfx_dp/cpewizard/User"
	Utility "github.com/ionosnetworks/qfx_dp/cpewizard/Utilities"
	LevelDb "github.com/syndtr/goleveldb/leveldb"
)

var (
	db LevelDb.DB
	//tokenTimer *time.Timer
	timerMap map[string]*(time.Timer)
)

// InsertValueWithKey With Given Key Value Pair
func InsertValueWithKey(Key []byte, Value []byte) {
	err := db.Put(Key, Value, nil)
	if err != nil {
		logr.Error("error inserting value in database", err.Error(), string(Key), string(Value))
	}
}

// CheckForGivenKey In Database
func CheckForGivenKeyAndReturnValue(key string) []byte {
	iter := db.NewIterator(nil, nil)
	for iter.Next() {
		currentKey := iter.Key()
		currentValue := iter.Value()
		if string(currentKey) == key {
			return currentValue
		}
	}
	iter.Release()
	err := iter.Error()
	if err != nil {
		logr.Error("Error releasing iterator", err.Error())
	}
	return nil
}

// CheckAndCreateDataBase at given Location
func CheckAndCreateDataBase(dbLocation string) {
	logr.Info("Creating DataBase")
	localDb, err := LevelDb.OpenFile(dbLocation, nil)
	if err != nil {
		logr.Error("Error creating database", err.Error())
		time.Sleep(60 * (time.Second))
		logr.Debug("Retrying to create DataBase")
		CheckAndCreateDataBase(Const.DataBaseLoc)
	} else {
		db = *localDb
	}
	timerMap = make(map[string]*time.Timer)
}

func CreateUserInfoObj() bool {
	var users []DataObj.UserInfoObj
	//ionosUser := DataObj.UserInfoObj{UserName: "admin@ionos.com", Password: Utility.GetHashValueOfString("ionos")}
	//users = append(users, ionosUser)
	usersJson, err := User.GetUsers()
	if err != nil {
		logr.Error("Error getting existing users", err.Error())
		ionosUser := DataObj.UserInfoObj{UserName: "debug@ionos.com", Password: Utility.GetHashValueOfString("ionos")}
		users = append(users, ionosUser)
	} else {
		currentUsers := usersJson.Users
		for i := 0; i < len(currentUsers); i++ {
			currUser := currentUsers[i]
			generalUser := DataObj.UserInfoObj{UserName: currUser.UserName, Password: currUser.Password}
			users = append(users, generalUser)
		}
	}
	byteData, err := Utility.ConvertObjectToByteArray(users)
	if err != nil {
		logr.Error("Error converting object", users, err.Error())
		return false
	} else {
		InsertValueWithKey([]byte(DbUserInfoKey), byteData)
	}
	return true
}

func UpdateDbUsers() {
	usersJson, err := User.GetUsers()
	if err != nil {
		logr.Error("Error getting existing users", err.Error())
		return
	}
	currUsers := GetUserInfo()
	for _, user := range usersJson.Users {
		userExists := false
		for i := 0; i < len(currUsers); i++ {
			currUser := currUsers[i]
			if currUser.UserName == user.UserName {
				userExists = true
				if currUser.Password != user.Password {
					currUser.Password = user.Password
					currUser.AuthToken = ""
					currUsers[i] = currUser
				}
			}
		}
		if !userExists {
			currUsers = append(currUsers, DataObj.UserInfoObj{UserName: user.UserName, Password: user.Password})
		}
	}

	for i := 0; i < len(currUsers); i++ {
		userRemoved := true
		currUser := currUsers[i]
		for _, user := range usersJson.Users {
			if currUser.UserName == user.UserName {
				userRemoved = false
			}
		}

		if userRemoved {
			currUsers = append(currUsers[:i], currUsers[i+1:]...)
			i--
		}
	}
	byteData, err := Utility.ConvertObjectToByteArray(currUsers)
	if err != nil {
		logr.Error("Error converting object", currUsers, err.Error())
	} else {
		InsertValueWithKey([]byte(DbUserInfoKey), byteData)
	}
}

func GetUserInfo() []DataObj.UserInfoObj {
	var usersInfo []DataObj.UserInfoObj
	userByteInfo := CheckForGivenKeyAndReturnValue(DbUserInfoKey)
	Utility.ConvertByteArrayToObject(userByteInfo, &usersInfo)
	return usersInfo
}

func UpdateUserInfoWithToken(authenToken string, userInfo DataObj.UserInfoObj) {
	currentUsersInfo := GetUserInfo()
	logr.Info("updating user info with data", userInfo)
	for i := 0; i < len(currentUsersInfo); i++ {
		currentUser := currentUsersInfo[i]
		if currentUser.UserName == userInfo.UserName && currentUser.Password == Utility.GetHashValueOfString(userInfo.Password) {
			currentUser.AuthToken = authenToken
			currentUsersInfo[i] = currentUser
		}
	}
	usrByteInfo, err := Utility.ConvertObjectToByteArray(currentUsersInfo)
	if err != nil {
		logr.Error("Error converting object", currentUsersInfo, err.Error())
	} else {
		InsertValueWithKey([]byte(DbUserInfoKey), usrByteInfo)
	}
}

func ClearAuthToken(token string) {
	currentUsersInfo := GetUserInfo()
	for i := 0; i < len(currentUsersInfo); i++ {
		currUser := currentUsersInfo[i]
		if currUser.AuthToken == token {
			currUser.AuthToken = ""
		}
		currentUsersInfo[i] = currUser
	}
	usrByteInfo, err := Utility.ConvertObjectToByteArray(currentUsersInfo)
	if err != nil {
		logr.Error("Error converting object", currentUsersInfo, err.Error())
	} else {
		InsertValueWithKey([]byte(DbUserInfoKey), usrByteInfo)
	}
}

func ClearAuthTokenAfterTime(duration uint, token string) {
	tokenTimer := time.NewTimer((time.Second) * (time.Duration(duration)))
	timerMap[token] = tokenTimer
	logr.Info("creating timer with duration", tokenTimer, duration)
	for {
		select {
		case <-tokenTimer.C:
			logr.Info("Current Auth Token Expired")
			ClearAuthToken(token)
		}
	}
}

func InvalidateTokenTimer(token string) {
	tokenTimer := timerMap[token]
	logr.Info("invalidatimg timer", tokenTimer)
	tokenTimer.Stop()
}
func getCurrentAuthTokens() ([]string, error) {
	var authTokens []string
	currentUsersInfo := GetUserInfo()
	logr.Info("current users data", currentUsersInfo)
	if currentUsersInfo == nil || len(currentUsersInfo) == 0 {
		return nil, errors.New("No Auth Token Found")
	}

	for _, currUser := range currentUsersInfo {
		authTokens = append(authTokens, currUser.AuthToken)
	}
	return authTokens, nil
}
func CreateCPInfoObj(cpeData DataObj.CPInfoObj) {
	byteCpData, err := Utility.ConvertObjectToByteArray(cpeData)
	if err != nil {
		logr.Error("Error converting object", cpeData, err.Error())
	} else {
		InsertValueWithKey([]byte(DbCpInfoKey), byteCpData)
	}
}

func GetCPInfo() DataObj.CPInfoObj {
	var cpeData DataObj.CPInfoObj
	cpByteData := CheckForGivenKeyAndReturnValue(DbCpInfoKey)
	Utility.ConvertByteArrayToObject(cpByteData, &cpeData)
	return cpeData
}

func CreatePortConfigs(portConfigs []DataObj.PortConfigInfo) {
	bytePortConfigs, err := Utility.ConvertObjectToByteArray(portConfigs)
	if err != nil {
		logr.Error("Error converting object", portConfigs, err.Error())
	} else {
		InsertValueWithKey([]byte(DbPortDetailsKey), bytePortConfigs)
	}
}

func GetPortConfigs() []DataObj.PortConfigInfo {
	var portConfigurations []DataObj.PortConfigInfo
	portConfigsByteData := CheckForGivenKeyAndReturnValue(DbPortDetailsKey)
	Utility.ConvertByteArrayToObject(portConfigsByteData, &portConfigurations)
	return portConfigurations
}

func UpdatePortsInfoWithNewInfo(updatedPortsInfo []DataObj.PortConfigInfo) error {
	existingPortConfigs := GetPortConfigs()
	for j := 0; j < len(updatedPortsInfo); j++ {
		var existingIndexForUpdatedConfig int = -1
		newPortConfig := updatedPortsInfo[j]
		for i := 0; i < len(existingPortConfigs); i++ {
			portConfig := existingPortConfigs[i]
			if portConfig.PrimaryLan == newPortConfig.PrimaryLan {
				existingIndexForUpdatedConfig = i
			}
		}
		existingPortConfigs[existingIndexForUpdatedConfig] = newPortConfig
	}
	newBytePortConfigs, err := Utility.ConvertObjectToByteArray(existingPortConfigs)
	if err != nil {
		logr.Error("Error converting object", existingPortConfigs, err.Error())
	} else {
		InsertValueWithKey([]byte(DbPortDetailsKey), newBytePortConfigs)
	}
	return nil
}

func DeleteCopyFlagForInterfaceFile() {
	err := db.Delete([]byte(DbIntfCopyFlag), nil)
	if err != nil {
	}
}

func GetCopyFlagForInterfaceFile() bool {
	var copyFlag bool
	copyFlagBytes := CheckForGivenKeyAndReturnValue(DbIntfCopyFlag)
	if copyFlagBytes == nil {
		return false
	}
	Utility.ConvertByteArrayToObject(copyFlagBytes, &copyFlag)
	return copyFlag
}

func UpdateFlagForInterfaceFileCopy(copyFlag bool) {
	flagBytes, err := Utility.ConvertObjectToByteArray(copyFlag)
	if err != nil {
	}
	InsertValueWithKey([]byte(DbIntfCopyFlag), flagBytes)
}

func ValidateAuthToken(authTok string) bool {
	var isValidTok bool
	tokens, err := getCurrentAuthTokens()
	logr.Info("current tokens", authTok, tokens)
	if err != nil {
		logr.Error("Error getting auth tokens")
		isValidTok = false
	}

	for _, token := range tokens {
		if token == authTok {
			isValidTok = true
		}
	}
	return isValidTok
}

//func CheckDnsResolution() error {
//	interfaces := GetPortConfigs()
//	for _, iface := range interfaces {
//		if len(iface.PrimaryDns) > 0 && !Utility.DnsResolution(iface.PrimaryDns) {
//			return errors.New(Const.PrimaryDnsFail)
//		} else if len(iface.SecondaryDns) > 0 && !Utility.DnsResolution(iface.SecondaryDns) {
//			return errors.New(Const.SecondaryDnsFail)
//		}
//	}
//	return nil
//}
