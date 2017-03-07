package dp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	// log "github.com/Sirupsen/logrus"
	//	"github.com/ionosnetworks/qfx_cmn/blog"
	p "github.com/ionosnetworks/qfx_dp/pipeline"
)

type Encryption struct{}

const (
	ICA_DP_ENCRYPTION_HEADER_SIZE      = 25
	ICA_DP_ENCRYPTION_BARE_HEADER_SIZE = 9
)

func (f *Encryption) Init() bool {
	return true
}

func SlapEncryptionHeader(headroom []byte, payloadLength uint64, version byte) {
	headroom[0] = version
	var i uint32
	for i = 1; i < 9; i++ {
		headroom[i] = byte((payloadLength &
			(0xff << ((i - 1) * 8))) >> ((i - 1) * 8))
	}
}

func (f *Encryption) Process(name string, req *p.ProcessReqResp) bool {
	if req.Interests[req.CurrentInterest] == "encrypt" {
		txreq := req.Data.(*TxJobReqResp)
		//		log.Debug(ctx, "Chunk encryption start for job",
		//			req.JobId, "chunk: ", txreq.cfo.ChunkNum)
		key := txreq.JobInfo.Key
		//		log.Debug(ctx, "Key length ", len(key))
		block, err := aes.NewCipher([]byte(key[0:32]))
		if err != nil {
			//	log.Err(ctx, "could not encrypt chunk",
			//		blog.Fields{"ChunkNum": txreq.cfo.ChunkNum,
			// "Job Id": req.JobId,
			//			"Err": err})
			return false
		}
		woff := txreq.cfo.WriteOffset
		iv := txreq.buffer[woff-aes.BlockSize : txreq.cfo.WriteOffset]
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			log.Err(ctx, "Error creating IV", nil)
			return false
		}
		buffer := txreq.buffer[woff:]
		/* Read a random aes.Blocksize string */
		encrypter := cipher.NewCFBEncrypter(block, iv)
		encrypter.XORKeyStream(buffer, buffer)
		var version byte = 1
		var payloadLength uint64 = txreq.cfo.ChunkActualSize +
			(txreq.cfo.ChunkActualSize % aes.BlockSize)
		SlapEncryptionHeader(
			txreq.buffer[woff-ICA_DP_ENCRYPTION_HEADER_SIZE:],
			payloadLength, version)
		txreq.cfo.WriteOffset -= ICA_DP_ENCRYPTION_HEADER_SIZE
		txreq.cfo.ChunkActualSize = payloadLength +
			ICA_DP_ENCRYPTION_HEADER_SIZE
			//		log.Debug(ctx, "Chunk encryption end for job",
			//			req.JobId, "chunk: ", txreq.cfo.ChunkNum)
		return true
	} else if req.Interests[req.CurrentInterest] == "decrypt" {
		rxreq := req.Data.(*RxJobReqResp)
		//		log.Debug(ctx, "Chunk decryption start for job",
		//			req.JobId, "chunk: ", rxreq.cfo.ChunkNum)
		key := rxreq.JobInfo.Key
		block, err := aes.NewCipher([]byte(key[0:32]))
		if err != nil {
			//	log.Err(ctx, "could not decrypt chunk",
			//		blog.Fields{"ChunkNum": rxreq.cfo.ChunkNum,
			// "Job Id": req.JobId,
			//			"Err": err})
			return false
		}
		encpheader := rxreq.buffer[0:ICA_DP_ENCRYPTION_HEADER_SIZE]
		buffer := rxreq.buffer[ICA_DP_ENCRYPTION_HEADER_SIZE:]
		iv := encpheader[ICA_DP_ENCRYPTION_BARE_HEADER_SIZE:]
		decrypter := cipher.NewCFBDecrypter(block, iv)
		decrypter.XORKeyStream(buffer, buffer)
		rxreq.buffer = buffer
		//		log.Debug(ctx, "Chunk decryption end for job",
		//			req.JobId, "chunk: ", rxreq.cfo.ChunkNum)
		return true
	}
	return false
}

func (f *Encryption) HeaderSpace() uint64 {
	return 9
}

func (f *Encryption) Exit() {
	log.Info(ctx, "Encryption state exited", nil)
}
