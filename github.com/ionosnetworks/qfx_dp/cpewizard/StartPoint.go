package main

import (
	"fmt"
	"log"
	"net/http"
	//"os"
	"strings"
	//"syscall"
	"time"

	logr "github.com/Sirupsen/logrus"
	//"github.com/ionosnetworks/ics/ica-dataplane/cp"
	Const "github.com/ionosnetworks/qfx_dp/cpewizard/Constants"
	DBManager "github.com/ionosnetworks/qfx_dp/cpewizard/DataBase"
	DataObj "github.com/ionosnetworks/qfx_dp/cpewizard/DataObjects"
	FileManager "github.com/ionosnetworks/qfx_dp/cpewizard/FileManager"
	CPRouter "github.com/ionosnetworks/qfx_dp/cpewizard/Routing"
	Utility "github.com/ionosnetworks/qfx_dp/cpewizard/Utilities"
	Sqlite3Util "github.com/ionosnetworks/qfx_dp/cpewizard/sqlite3Util"
	//Walk "github.com/ionosnetworks/qfx_dp/walk"
)

func InitializeDependencies() {
	//Init_Logger(Const.LogFileLocation, Const.LogFileLevel)
	DBManager.CheckAndCreateDataBase(Const.DataBaseLoc)
	if !DBManager.CreateUserInfoObj() {
		time.Sleep(30 * (time.Second))
		logr.Debug("Retrying cpe user creation")
		DBManager.CreateUserInfoObj()
	}
	FileManager.CreateCPData(Utility.GetIc2Connectivity())
	DBManager.CreatePortConfigs(FileManager.GetInterfaces())
	FileManager.CreateLogTarFile(Const.LogFileLocation)
	//FileManager.CreateSoftLinkForFile(Const.LogFileLocation, Const.LogFileSoftLink)
}
func createEthernetMapping() {
	Const.EthernetMapping = make(map[string]string)
	mappingBytes := FileManager.ReadFile(Const.EthMapFile)
	if mappingBytes != nil {
		mappingData := Utility.RemoveEmptyStringsFromArray(strings.Split(string(mappingBytes), "\n"))
		for _, currentEthConf := range mappingData {
			ethPair := strings.Split(currentEthConf, ":")
			Const.EthernetMapping[ethPair[1]] = "Ethernet-" + ethPair[0]
		}
		lastEth := mappingData[len(mappingData)-1]
		eths := strings.Split(lastEth, ":")
		Const.NonEditablePhysicalEth = "Ethernet" + eths[0]
		Const.NonEditableLogicalEth = eths[1]
	}

	logr.Info("Non editable logical and physical eth", Const.NonEditableLogicalEth, Const.NonEditablePhysicalEth, Const.EthernetMapping)
}
func InitCpeWizard() {
	//Utility.WizCpChannel = WizCpChan
	//Utility.CpWizChannel = CpWizChan
	cpeRouter := CPRouter.CreateNewRouter() //mux.NewRouter()
	InitializeDependencies()
	createEthernetMapping()
	cpeRouter.HandleFunc("/", serverStarted)
	log.Fatal(http.ListenAndServe(":8080", cpeRouter))
}

func saveFilesToDB(fileListChannel chan DataObj.File, done chan bool) {
	fileList := []DataObj.File{}
	f := DataObj.File{}

	for {
		dbwrite := false
		more := true
		select {
		case <-time.After(100 * time.Millisecond):
			dbwrite = true

		case f, more = <-fileListChannel:
			if more {
				fileList = append(fileList, f)
			}
			if len(fileList) == 100 || more == false {
				dbwrite = true
			}
		}
		if dbwrite && len(fileList) > 0 {
			Sqlite3Util.StoreFileList(fileList)
			fileList = []DataObj.File{}
		}
		if more == false {
			fmt.Println("Done writing!!")
			done <- true
			return
		}
	}
}

func InitializeSqliteDB() {
	startTime := time.Now()
	Sqlite3Util.CreateSqliteDB()
	/*fileListChannel := make(chan DataObj.File, 1000)
	done := make(chan bool)
	go saveFilesToDB(fileListChannel, done)
	Sqlite3Util.StoreLogInfo(startTime, startTime, "Processing")
	Walk.Walk(Const.CpeRootDir, func(path string, f os.FileInfo, err error) error {
		if Const.CpeRootDir != path {
			fi := DataObj.File{0, strings.TrimRight(path, f.Name()), f.Name(), f.Size(), f.IsDir(), false, strings.Count(path, "/"), f.ModTime()}
			fileListChannel <- fi
		}
		return nil
	})
	close(fileListChannel)
	<-done
	syscall.Sync()
	Sqlite3Util.UpdateLogInfo(startTime, time.Now(), "Done")*/
	duration := time.Since(startTime)
	fmt.Println("Total Time Taken", duration.Seconds())
}

func main() {
	cpeRouter := CPRouter.CreateNewRouter() //mux.NewRouter()
	InitializeDependencies()
	createEthernetMapping()
	InitializeSqliteDB()
	cpeRouter.HandleFunc("/", serverStarted)
	log.Fatal(http.ListenAndServe(":8080", cpeRouter))
}

func serverStarted(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello there mate\n")
}
