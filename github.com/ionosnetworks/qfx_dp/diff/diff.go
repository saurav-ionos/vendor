package main

import (
	"fmt"
	"sync"
	"time"
)

type fileInfo struct {
	size  uint64
	mtime time.Time
}

type fileTable struct {
	m map[string]fileInfo
	*sync.Mutex
}

var ft fileTable

func (f1 fileInfo) equals(f2 fileInfo) (ok bool) {
	if f1.size == f2.size && f1.mtime.Equal(f2.mtime) {
		ok = true
	} else {
		ok = false
	}
	return
}

func computeDiff(ft2 fileTable) {
	for k, v := range ft.m {
		val, ok := ft2.m[k]
		if ok != false {
			if val.equals(v) {
				fmt.Println("Equals")
			} else {
				fmt.Println("Unequal")
			}
		} else {
			fmt.Println("Unequal")
		}
	}
}

func main() {
	ft.m = make(map[string]fileInfo)
	ft.Mutex = new(sync.Mutex)
	var ft2 fileTable
	ft2.m = make(map[string]fileInfo)
	ft2.Mutex = new(sync.Mutex)
	var f1, f2 fileInfo
	f1.size = 123
	f1.mtime = time.Now()
	f2.size = 123
	f2.mtime = time.Now() //f1.mtime //time.Now()
	ft.Lock()
	ft.m["123"] = f1
	ft.Unlock()
	ft2.Lock()
	ft2.m["123"] = f2
	ft2.Unlock()
	fmt.Println(f1.mtime)
	fmt.Println(f2.mtime)
	computeDiff(ft2)
}
