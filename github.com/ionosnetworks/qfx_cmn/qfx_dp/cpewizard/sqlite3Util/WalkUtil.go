package sqlite3Util

import (
	"fmt"
	DataObj "github.com/ionosnetworks/qfx_dp/cpewizard/DataObjects"
	Walk "github.com/ionosnetworks/qfx_dp/walk"

	"os"
	"strings"
	"syscall"
	"time"
)

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
			StoreFileList(fileList)
			fileList = []DataObj.File{}
		}
		if more == false {
			fmt.Println("Done writing!!")
			done <- true
			return
		}
	}
}

func WalkAndUpdateDB(dirPath string) {
	fmt.Println("Walk and Update: ", dirPath)

	startTime := time.Now()
	fileListChannel := make(chan DataObj.File, 1000)
	done := make(chan bool)
	go saveFilesToDB(fileListChannel, done)
	StoreLogInfo(startTime, startTime, "Processing")
	Walk.Walk(dirPath, func(path string, f os.FileInfo, err error) error {
		if dirPath != path {
			fi := DataObj.File{0, strings.TrimRight(path, f.Name()), f.Name(), f.Size(), f.IsDir(), false, strings.Count(path, "/"), f.ModTime()}
			fileListChannel <- fi
		}
		return nil
	})
	close(fileListChannel)
	<-done
	syscall.Sync()
	UpdateLogInfo(startTime, time.Now(), "Done")
	duration := time.Since(startTime)
	fmt.Println("Total Time Taken", duration.Seconds(), " to update the path: ", dirPath)
}
