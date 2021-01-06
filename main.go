package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dimuls/classifier/bayesian"
	"github.com/dimuls/classifier/mystem"
	"github.com/dimuls/classifier/web"
	"github.com/sirupsen/logrus"
)

type webClassifiersPoolAdapter struct {
	*bayesian.ClassifiersPool
}

func (a webClassifiersPoolAdapter) Classifier(classifierID string,
	create bool) (c web.Classifier, exists bool) {
	return a.ClassifiersPool.Classifier(classifierID, create)
}

func main() {
	logrus.SetLevel(logrus.DebugLevel)

	mt := mystem.NewTokenizer(os.Getenv("MYSTEM_TOKENIZER_BIN_PATH"))

	bcp, err := bayesian.NewClassifiersPool(mt,
		os.Getenv("BAYESIAN_CLASSIFIERS_POOL_DATA_PATH"))
	if err != nil {
		logrus.WithError(err).Error(
			"failed to create bayesian classifier")
	}

	ws := web.NewServer(webClassifiersPoolAdapter{bcp},
		os.Getenv("WEB_SERVER_BIND_ADDR"),
		os.Getenv("WEB_SERVER_DEBUG") == "1")

	ws.Start()

	signals := make(chan os.Signal)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	logrus.Infof("captured %v signal, stopping", <-signals)

	st := time.Now()

	ws.Stop()

	logrus.Infof("stopped in %s seconds, exiting", time.Now().Sub(st))
}
