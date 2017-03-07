package dp

import (
	"github.com/ionosnetworks/qfx_cmn/blog"
	"os"
	"path"
	"strconv"
	"strings"
)

/*
Don't change the order of the various checks being done in this function
Think thrice before you change the order :)
*/
func isFileGood(fileName string, isDir bool,
	fileSize uint64, modTime string) bool {
	fi, err := os.Stat(fileName)
	if err != nil {
		log.Err(ctx, "Error in isFileGood", blog.Fields{"Err": err})
		return false
	}
	if fi.IsDir() != isDir {
		log.Err(ctx, "Mismatch in file/dir type of",
			blog.Fields{"name": fileName,
				"Provided Dir=": isDir,
				"Found Dir=":    fi.IsDir()})
		return false
	}
	mtimeInt, err := strconv.ParseInt(modTime, 10, 64)
	//	log.Debug(ctx, "mtimeint=", mtimeInt, "fi modtime=", fi.ModTime().Unix())
	if fi.ModTime().Unix() != mtimeInt {
		log.Err(ctx, "Mismatch in file modtime ",
			blog.Fields{"name": fileName,
				"Provided=": mtimeInt,
				"Found=":    fi.ModTime().Unix()})
		return false
	}
	if isDir {
		return true
	}
	if fi.Size() != int64(fileSize) {
		log.Err(ctx, "Mismatch in file size of",
			blog.Fields{"name": fileName,
				"Provided=": fileSize,
				"Found=":    fi.Size()})
		return false
	}
	return true
}

func getchunkElts(storMsg string) []string {
	var s string = strings.Fields(storMsg)[1]
	chunkName := path.Base(s)
	xt := strings.Split(chunkName, "-")
	return xt
}
func getJobId(storMsg string) uint32 {
	xt := getchunkElts(storMsg)
	syncID := strings.Trim(xt[0], "syncID")
	i, err := strconv.Atoi(syncID)
	if err == nil {
		return uint32(i)
	} else {
		return 0xFFFFFFFF
	}
}

func getChunkNumber(storMsg string) uint32 {
	xt := getchunkElts(storMsg)
	i, err := strconv.Atoi(xt[1])
	if err == nil {
		return uint32(i)
	} else {
		return 0xFFFFFFFF
	}
}

func getAdvertizedSize(storMsg string) uint32 {
	xt := getchunkElts(storMsg)
	to_convert := strings.Split(xt[3], ".")
	i, err := strconv.Atoi(to_convert[0])
	if err == nil {
		return uint32(i)
	} else {
		log.Err(ctx, "Error converting number",
			blog.Fields{"Err": err})
		return 0xFFFFFFFF
	}
}

func getChunkPath(storMsg string) string {
	name := storMsg[5:]
	return name
}

func getChunkInfo(chunkInfo *ChunkInfo, storMsg string) {
	storstr := string(storMsg)
	//	chunkInfo.ChunkNum = getChunkNumber(storstr)
	//	chunkInfo.ChunkAdvertizedSize = getAdvertizedSize(storstr)
	chunkInfo.ChunkPath = getChunkPath(storstr)
}
