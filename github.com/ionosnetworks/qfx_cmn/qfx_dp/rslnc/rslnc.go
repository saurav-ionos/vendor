package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"github.com/boltdb/bolt"
	"os"
	//	"github.com/ionosnetworks/qfx_dp/infra"
	"sync"
)

type UUID [16]byte

type JobletMod struct {
	StartOffset uint64
	EndOffset   uint64
	// contribution of this joblet to the size of the job
	Size uint64
}

type JobletChunkInfo struct {
	JobletId      uint32
	Mod           []JobletMod
	JobletCorrupt bool
	Forder        uint32
}

type rslncMap struct {
	m map[UUID][]JobletChunkInfo
	sync.Mutex
}

var rMap rslncMap
var db *bolt.DB

func SaveSync(syncID uint32, uuid UUID, info []JobletChunkInfo) {
	rMap.Lock()
	rMap.m[uuid] = info
	rMap.Unlock()
	//TODO open DB and dump contents of rMap in separate buckets for
	// separamainte SyncIDs
	err := db.Update(func(tx *bolt.Tx) error {
		bucketID := make([]byte, 4)
		binary.LittleEndian.PutUint32(bucketID, syncID)
		bucket, err := tx.CreateBucketIfNotExists(bucketID)
		if err != nil {
			return err
		}
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		err = enc.Encode(info)
		if err != nil {
			fmt.Println("Encode error", err)
			return err
		}
		value := buf.Bytes()
		key := uuid[:]

		err = bucket.Put(key, value)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		fmt.Println("Failed inserting to DB", err.Error())
	}
}
func GenUUID() UUID {
	f, _ := os.Open("/dev/urandom")
	var b UUID
	f.Read(b[:])
	f.Close()
	return b
}

func main() {
	fmt.Println("Initiating resiliency")
	rMap.m = make(map[UUID][]JobletChunkInfo)
	db, err := bolt.Open("bolt.db", 0644, nil)
	// defer db.Close()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("DB opened")
	var val []JobletChunkInfo
	inf := JobletChunkInfo{
		JobletId: 1,
		Mod:      nil,
		Forder:   1,
	}
	val = append(val, inf)
	uuid := GenUUID()
	SaveSync(100, uuid, val)

}
