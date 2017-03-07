// This package provides a REST API frontend to bolt db/leveldb
package slcemulator

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

type NhInfo struct {
	Name  string `json:"NAME"`
	SlcIP string `json:"SLCIP"`
}

const nhPath string = "/var/ionos/nh.txt"

var rentry NhInfo

func init() {

	fi, err := os.Open(nhPath)
	if err != nil {
		panic(err)
	}
	err = json.NewDecoder(fi).Decode(&rentry)
	if err != nil {
		panic(err)
	}
	defer fi.Close()

}

func getNextHopRegion(w http.ResponseWriter, req *http.Request) {

	json.NewEncoder(w).Encode(rentry)
	return
}

func Start() {
	router := mux.NewRouter()
	router.HandleFunc("/{id}", getNextHopRegion).Methods("GET")
	go func() {
		s := &http.Server{
			Addr:           ":12345",
			Handler:        router,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}
		fmt.Println("Starting slc emulator...")
		s.ListenAndServe()
	}()
}
