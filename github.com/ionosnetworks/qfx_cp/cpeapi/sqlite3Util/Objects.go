package sqlite3Util

import (
	"time"
)

type File struct {
	FilePath   string
	FileName   string
	FileSize   int64
	IsDir      bool
	IsExported bool
	Level      int
	ModTime    time.Time
}
