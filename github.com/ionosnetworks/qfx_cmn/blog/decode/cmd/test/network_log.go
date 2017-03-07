package main

import (
	"fmt"
	"time"

	"github.com/ionosnetworks/qfx_cmn/blog"
)

var ctx = "new-context"

func main() {
	//fo, err := net.Dial("tcp", "127.0.0.1:"+os.Args[1])
	//if err != nil {
	//	panic(err)
	//}
	//logger := blog.New(fo)
	logger := blog.New("0.0.0.0:3000", "", "")
	//logger := blog.LazyLog("0.0.0.0:3000", "", "")
	fmt.Println("current logger", logger)
	logger.SetLevel(blog.Debug)

	for i := 0; i <= 1; i++ {
		logger.Debug(ctx, "this is a debug log", blog.Fields{"a": 10, "b": 11})
		logger.Info(ctx, "this is a info log", nil)
		logger.Warn(ctx, "this is a warning log", nil)
		logger.Warn(ctx, "this is a warning log", nil)
		logger.Warn(ctx, "this is a warning log", nil)
		logger.Warn(ctx, "this is a warning log", nil)
		logger.Err(ctx, "this is a error log", nil)
		logger.Crit(ctx, "this is a critical log", nil)
	}
	time.Sleep(8 * time.Second)
	logger.Warn(ctx, "this is a repeat log", nil)
	logger.Err(ctx, "this is a repeat1 log", nil)
	logger.Crit(ctx, "this is a repeat2 log", nil)
	logger.Crit(ctx, "this is a repeat3 log", nil)
	//time.Sleep(10 * time.Second)
	logger.Close()
	return

}
