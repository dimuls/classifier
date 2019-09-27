package classifier

import (
	"os"

	"github.com/dimuls/classifier/mystem"
	"github.com/dimuls/classifier/web"
)

type Service struct {
	classifier         *Classifier
	classifierFilePath string
	webServer          *web.Server
}

func NewService(mystemBinPath string, classifierFilePath string,
	webServerBindAddr string, webServerDebug bool) (*Service, error) {

	c := NewClassifier(mystem.NewWordsExtractor(mystemBinPath))

	s := &Service{
		classifier:         c,
		classifierFilePath: classifierFilePath,
		webServer:          web.NewServer(webServerBindAddr, c, webServerDebug),
	}

	classifierFileStat, err := os.Stat(classifierFilePath)
	if !os.IsNotExist(err) && !classifierFileStat.IsDir() {
		c.Load(classifierFilePath)
	}

	return s, nil
}

func (s *Service) Start() {
	s.webServer.Start()
}

func (s *Service) Stop() {
	s.webServer.Stop()
	s.classifier.Save(s.classifierFilePath)
}
