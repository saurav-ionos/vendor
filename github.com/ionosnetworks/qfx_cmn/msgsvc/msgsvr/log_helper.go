package main

import (
	"fmt"
	"os"

	"github.com/ionosnetworks/qfx_cmn/blog"
)

func InitLogger(accessKey, secret string) {

	logSvr := "127.0.0.1:2000"
	if val := os.Getenv("LOG_SERVER"); val != "" {
		logSvr = val
	}

	if logger = blog.New(logSvr, accessKey, secret); logger == nil {
		fmt.Println("Logger failed ")
		return
	}

	logLevel := "Debug"
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		logLevel = level
	}

	switch logLevel {
	case "Debug":
		logger.SetLevel(blog.Debug)
	case "Info":
		logger.SetLevel(blog.Info)
	case "Warn":
		logger.SetLevel(blog.Warn)
	case "Err":
		logger.SetLevel(blog.Err)
	case "Crit":
		logger.SetLevel(blog.Crit)
	}
}
