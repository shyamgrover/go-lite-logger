package main

import (
	"github.com/shyamgrover/go-lite-logger/logWriter"
	"github.com/shyamgrover/go-lite-logger/logger"
)

func main() {
	var myLogger1 *logger.Logger
	errorCallback := func() {
		myLogger1.CloseLogger()
	}
	myLogger1, err := logger.CreateLogger(logWriter.InfoLevel, "myLogger.log", "", errorCallback)
	if err == nil {
		for i := 0; i < 500; i++ {
			myLogger1.Info(i)
		}
	}
	myLogger1.CloseLogger()
}
