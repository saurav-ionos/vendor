package main

import (
	"database/sql"
	"fmt"
	Sqlite3Util "github.com/ionosnetworks/qfx_cp/cpeapi/sqlite3Util"
	Walk "github.com/ionosnetworks/qfx_cp/cpeapi/walk"
	"os"
	"strings"
	"syscall"
	"time"
)

func main() {
	startTime := time.Now()
	searchDir := "/home/niti/Desktop/test"
	db := Sqlite3Util.InitDB("cpeData.db")
	Sqlite3Util.CreateTable(db)
	fileList := []Sqlite3Util.File{}
	if Sqlite3Util.CheckLastProcess(db) != "Done" {
		Sqlite3Util.ClearTable(db)
		fmt.Println("Rootpath: ", searchDir)
		fmt.Println("Database Tables created!")
		fileListChannel := make(chan Sqlite3Util.File, 1000)
		done := make(chan bool)
		go saveFilesToDB(db, fileListChannel, done)
		Sqlite3Util.StoreLogInfo(db, startTime, startTime, "Processing")
		Walk.Walk(searchDir, func(path string, f os.FileInfo, err error) error {
			fi := Sqlite3Util.File{strings.TrimRight(path, f.Name()), f.Name(), f.Size(), f.IsDir(), false, strings.Count(path, "/"), f.ModTime()}
			fileListChannel <- fi
			return nil
		})
		close(fileListChannel)
		<-done
		syscall.Sync()
		Sqlite3Util.UpdateLogInfo(db, startTime, time.Now(), "Done")
		db.Close()
		duration := time.Since(startTime)
		fmt.Println("Total Time Taken", duration.Seconds())
	} else {
		fileList = Sqlite3Util.GetRootDirs(db)
		fmt.Println(len(fileList))
		fmt.Println(Sqlite3Util.GetDirCount(db, "/home/niti/Desktop/test/"))
		fileList = Sqlite3Util.GetDirInfo(db, "")
	}

}

func saveFilesToDB(db *sql.DB, fileListChannel chan Sqlite3Util.File, done chan bool) {
	fileList := []Sqlite3Util.File{}
	var f Sqlite3Util.File
	for {
		dbwrite := false
		more := true
		select {
		case <-time.After(2 * time.Second):
			dbwrite = true

		case f, more = <-fileListChannel:
			fileList = append(fileList, f)
			if len(fileList) == 100 || more == false {
				dbwrite = true
			}
		}
		if dbwrite && len(fileList) > 0 {
			Sqlite3Util.StoreFileList(db, fileList)
			fileList = []Sqlite3Util.File{}
		}
		if more == false {
			fmt.Println("Done writing!!")
			done <- true
			return
		}

	}
}
