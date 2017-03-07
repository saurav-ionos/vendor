package main

import (
	"os"

	"github.com/ionosnetworks/qfx_cmn/blog"
)

var ctx = "new-context"

func main() {
	fo, err := os.OpenFile("test.log", os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	logger := blog.New(fo)
	logger.SetLevel(blog.Debug)

	for i := 0; i <= 1; i++ {
		logger.Debug(ctx, "this is a debug log", blog.Fields{"a": 10, "b": 11})
		logger.Info(ctx, "this is a info log", nil)
		logger.Warn(ctx, "this is a warning log", nil)
		logger.Err(ctx, "this is a error log", nil)
		logger.Crit(ctx, "this is a critical log", nil)
	}

	logger.Close()

	return

}
