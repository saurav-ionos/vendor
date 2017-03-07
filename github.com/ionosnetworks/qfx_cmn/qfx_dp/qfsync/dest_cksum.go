package qfsync

import (
	"crypto/sha1"
	"sync"
)

type csumReq struct {
	data []byte
}

var csumChan chan csumReq

//const csumSize = md5.Size
const csumSize = sha1.Size

func doBatchCksum(blocksize int, data []byte) [][csumSize]byte {
	var wg sync.WaitGroup
	numBlocks, rem := len(data)/blocksize, len(data)%blocksize
	if rem > 0 {
		numBlocks++
	}
	result := make([][csumSize]byte, numBlocks)

	if numBlocks < 4 {
		wg.Add(1)
		go doBlockCksum(&wg, blocksize, data, result)
		wg.Wait()
		return result
	}

	// We would want to have atleast 1 block in a batch
	blocksPerBatch := numBlocks / 4
	indexAtBatchEnd := blocksPerBatch * blocksize

	dataBatch1 := data[0:indexAtBatchEnd]
	result1 := result[0:blocksPerBatch]
	dataBatch2 := data[indexAtBatchEnd : 2*indexAtBatchEnd]
	result2 := result[blocksPerBatch : 2*blocksPerBatch]
	dataBatch3 := data[2*indexAtBatchEnd : 3*indexAtBatchEnd]
	result3 := result[2*blocksPerBatch : 3*blocksPerBatch]
	dataBatch4 := data[3*indexAtBatchEnd:]
	result4 := result[3*blocksPerBatch:]
	// checksums
	wg.Add(4)
	go doBlockCksum(&wg, blocksize, dataBatch1, result1)
	go doBlockCksum(&wg, blocksize, dataBatch2, result2)
	go doBlockCksum(&wg, blocksize, dataBatch3, result3)
	go doBlockCksum(&wg, blocksize, dataBatch4, result4)
	wg.Wait()
	return result
}

func doBlockCksum(wg *sync.WaitGroup, blockSize int,
	batchData []byte, result [][csumSize]byte) {

	// Keep generating the md5 checksum at block boundaries
	for i := 0; ; i++ {
		var end int
		var exit bool
		if len(batchData) > blockSize {
			end = blockSize
		} else {
			end = len(batchData)
			exit = true
		}
		sumData := batchData[0:end]
		//result[i] = md5.Sum(sumData)
		result[i] = sha1.Sum(sumData)
		if exit {
			break
		}
		batchData = batchData[blockSize:]
	}
	wg.Done()
}
