// A parallel version of file walker
package walk

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
)

type WalkFunc func(filepath string, fi os.FileInfo, err error) error

type dirobj struct {
	Path      string
	Work      WalkFunc
	WaitGroup *sync.WaitGroup
}

var dirObjPool sync.Pool

func init() {
	dirObjPool.New = func() interface{} {
		return new(dirobj)
	}
}
func directoryWorker(workerid int, dirchan chan *dirobj,
	worwg *sync.WaitGroup) {
	defer worwg.Done()
	for job := range dirchan {
		readDirectory(job, dirchan)
	}
}
func createDirWorkerPool(poolsize int, worwg *sync.WaitGroup,
	dirchan chan *dirobj) {
	for w := 0; w < poolsize; w++ {
		worwg.Add(1)
		go directoryWorker(w, dirchan, worwg)
	}
}

func readDirectory(job *dirobj, dirchan chan *dirobj) {
	defer job.WaitGroup.Done()
	files, err := ioutil.ReadDir(job.Path)
	if err != nil {
		job.Work(job.Path, nil, err)
	}
	for _, file := range files {
		erro := job.Work(filepath.Join(job.Path, file.Name()), file, nil)
		if file.IsDir() && erro != nil && erro == filepath.SkipDir {
			continue
		} else if file.IsDir() {
			newjob := dirObjPool.Get().(*dirobj)
			newjob.Path = filepath.Join(job.Path, file.Name())
			newjob.Work = job.Work
			newjob.WaitGroup = job.WaitGroup
			job.WaitGroup.Add(1)
			select {
			case dirchan <- newjob:
			default:
				readDirectory(newjob, dirchan)
			}
		}
	}
	dirObjPool.Put(job)
}

func Walk(root string, work WalkFunc) error {

	info, err := os.Lstat(root)
	if err != nil {
		return work(root, info, err)
	}
	var waitForWorkers sync.WaitGroup
	var dirCount sync.WaitGroup
	dirchan := make(chan *dirobj, 1024)
	createDirWorkerPool(16, &waitForWorkers, dirchan)
	dirCount.Add(1)
	rootDirObj := &dirobj{Path: root, Work: work, WaitGroup: &dirCount}
	dirchan <- rootDirObj //rootFolder
	dirCount.Wait()
	close(dirchan)
	waitForWorkers.Wait()
	return nil
}
