package qfsync

import (
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
	"github.com/ionosnetworks/qfx_dp/walk"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	orderTypeOrdering int = iota
	orderTypeMetaCollecion
)

type Focb func(f FileOrder, startIndex int64, endIndex int64) error
type fileOrder struct {
	db         *leveldb.DB // The leveldb handle where the file ordering resider
	dbLock     sync.RWMutex
	batchSize  int32
	fpath      string
	root       string
	focb       Focb
	idx        int64
	matchTable map[[md5.Size]byte]struct{}
	lastIdx    int64
}

type metaPair struct {
	path string
	info os.FileInfo
}

type metaInfo struct {
	Path    string
	ModTime time.Time
	Size    int64
}

type FileOrder interface {
	Order() (int64, error)
	GetFileOrder(startIndex, endIndex int64) ([]string, error)
	GetFileNameAtIdx(indx int64) (string, error)
	SetFileOrder(startIndex, endIndex int64, l []string) error
	GetLastIdx() int64
}

// buildNewFileOrder builds a random file ordering for the provided
// overwrite flag will replace the file ordering db if set and available
// for every batchSize number of file focb is called passing the startIndex
// and the endIndex in the batch
func buildNewFileOrder(root string, batchSize int32, orderPrefix string,
	focb Focb) (*fileOrder, error) {
	fileOrder := new(fileOrder)
	fileOrder.batchSize = batchSize
	fileOrder.focb = focb
	fileOrder.root = root
	orderPath := fmt.Sprintf("%s/ordering", orderPrefix)
	db, err := leveldb.OpenFile(orderPath, nil)
	if err != nil {
		panic(err)
	}
	fileOrder.db = db
	fileOrder.matchTable = make(map[[md5.Size]byte]struct{})
	return fileOrder, err

}

func (f *fileOrder) callbackOnTrigger(ch chan metaPair) {
	var lastBatchEnd int64 = f.idx
	for {
		select {
		case p, ok := <-ch:
			// Channel is closed. Trigger callback if anything remains
			if !ok {
				if f.focb != nil && f.idx != lastBatchEnd {
					f.focb(f, lastBatchEnd+1, f.idx)
				}
				logger.DebugS("test-sync",
					"Exiting out of cbOntriggerLoop")
				return
			}
			if p.info.IsDir() {
				// nothing to do just continue
				continue
			}
			// Check if any ordering for the file
			// is present
			if _, ok := f.matchTable[md5.Sum([]byte(p.path))]; ok == true {
				//Ignore this
				logger.Debug("test-sync", "Ignoring entry for",
					blog.Fields{
						"filename": p.path})
				continue
			}
			f.idx++
			// Create the leveldb entry for it
			err := f.db.Put([]byte(fmt.Sprintf("%d", f.idx)),
				[]byte(p.path), nil)
			if err != nil {
				logger.Err("test-ctx",
					"error updating leveldb",
					blog.Fields{
						"fileid":   f.idx,
						"filename": p.path})
				continue
			}
			f.matchTable[md5.Sum([]byte(p.path))] = struct{}{}
			if f.focb != nil && f.idx%int64(f.batchSize) == 0 {
				f.focb(f, lastBatchEnd+1, f.idx)
				lastBatchEnd = f.idx
			}

		}
	}
}

func (f *fileOrder) Order() (int64, error) {

	var err error = nil
	triggerChan := make(chan metaPair)
	go f.callbackOnTrigger(triggerChan)

	logger.Debug("test-sync", "walk on", blog.Fields{
		"root": f.root})
	err = walk.Walk(f.root,
		func(path string, info os.FileInfo, err error) error {
			//triggerChan <- struct{}{}
			relpath, _ := filepath.Rel(f.root, path)
			triggerChan <- metaPair{
				path: relpath,
				info: info,
			}
			return nil
		})
	close(triggerChan)
	return f.idx, err

}

// GetFileOrder returns the list of files between the startIndex and the
// endIndex
func (f *fileOrder) GetFileOrder(startIndex, endIndex int64) ([]string, error) {
	fileOrder := make([]string, (endIndex - startIndex + 1))
	for i := startIndex; i <= endIndex; i++ {
		idx := fmt.Sprintf("%d", i)
		f.dbLock.RLock()
		s, err := (f.db.Get([]byte(idx), nil))
		f.dbLock.RUnlock()
		if err != nil {
			logger.ErrS("qfx_dp-fileorder", "error for key "+idx+" "+err.Error())
			return nil, err
		} else {
			fileOrder[i-startIndex] = string(s)
		}
	}
	return fileOrder, nil
}

func (f *fileOrder) GetFileNameAtIdx(idx int64) (string, error) {
	v, e := f.GetFileOrder(idx, idx)
	if e != nil {
		return "", e
	}
	return fmt.Sprintf("%s/%s", f.root, v[0]), nil
}

func (f *fileOrder) SetFileOrder(startIndex, endIndex int64, l []string) error {
	var err error
	for i := startIndex; i <= endIndex; i++ {
		f.dbLock.Lock()
		err = f.db.Put([]byte(fmt.Sprintf("%d", i)),
			[]byte(l[i-startIndex]), nil)
		f.dbLock.Unlock()
		if err != nil {
			return err
		}
	}
	if endIndex > f.idx {
		f.idx = endIndex
	}
	return err
}

func (f *fileOrder) GetLastIdx() int64 {
	return f.idx
}
