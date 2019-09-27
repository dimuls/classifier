package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/dimuls/classifier"
)

func main() {

	service, err := classifier.NewService(
		os.Getenv("MYSTEM_BIN_PATH"),
		os.Getenv("CLASSIFIER_FILE_PATH"),
		os.Getenv("WEB_SERVER_BIND_ADDR"),
		os.Getenv("WEB_SERVER_DEBUG") == "1")
	if err != nil {
		logrus.WithError(err).Fatal("failed to create classifier service")
	}

	service.Start()

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	signal := <-signals

	logrus.Infof("captured %v signal, stopping", signal)

	startTime := time.Now()

	service.Stop()

	endTime := time.Now()

	logrus.Infof("stopped in %g seconds, exiting",
		endTime.Sub(startTime).Seconds())
}
