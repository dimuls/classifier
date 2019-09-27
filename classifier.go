package classifier

import (
	"errors"
	"sync"

	"github.com/jbrukh/bayesian"
	"github.com/sirupsen/logrus"

	"github.com/dimuls/classifier/entity"
)

type WordsExtractor interface {
	ExtractWords(text string) ([]string, error)
}

type Classifier struct {
	wordsExtractor  WordsExtractor
	classifier      *bayesian.Classifier
	classifierMutex sync.RWMutex

	log *logrus.Entry
}

func NewClassifier(we WordsExtractor) *Classifier {
	return &Classifier{
		wordsExtractor: we,
		log:            logrus.WithField("subsystem", "classifier"),
	}
}

func (c *Classifier) Train(docs []entity.Document) {
	classesMap := map[string]struct{}{}
	for _, d := range docs {
		classesMap[d.Class] = struct{}{}
	}

	var classes []bayesian.Class
	for class := range classesMap {
		classes = append(classes, bayesian.Class(class))
	}

	classifier := bayesian.NewClassifier(classes...)

	for _, d := range docs {
		words, err := c.wordsExtractor.ExtractWords(d.Text)
		if err != nil {
			c.log.WithError(err).Error(
				"failed to extract words from document text")
			return
		}
		classifier.Learn(words, bayesian.Class(d.Class))
	}

	c.classifierMutex.Lock()
	c.classifier = classifier
	c.classifierMutex.Unlock()
}

func (c *Classifier) Trained() bool {
	c.classifierMutex.RLock()
	defer c.classifierMutex.RUnlock()

	return c.classifier != nil
}

func (c *Classifier) Classify(text string) (string, error) {
	c.classifierMutex.RLock()
	if c.classifier == nil {
		c.classifierMutex.RUnlock()
		return "", errors.New("classifier is not trained")
	}
	c.classifierMutex.RUnlock()

	words, err := c.wordsExtractor.ExtractWords(text)
	if err != nil {
		return "", errors.New("failed to extract words from text: " + err.Error())
	}

	if len(words) == 0 {
		return "", errors.New("no words extracted from text")
	}

	c.classifierMutex.RLock()
	_, i, _ := c.classifier.LogScores(words)
	c.classifierMutex.RUnlock()

	return string(c.classifier.Classes[i]), nil
}

func (c *Classifier) Save(path string) {
	c.classifierMutex.RLock()
	defer c.classifierMutex.RUnlock()
	if c.classifier == nil {
		return
	}
	err := c.classifier.WriteToFile(path)
	if err != nil {
		c.log.WithError(err).Error("failed to save classifier to file")
	}
}

func (c *Classifier) Load(path string) {
	classifier, err := bayesian.NewClassifierFromFile(path)
	if err != nil {
		c.log.WithError(err).Error("failed to load classifier from file")
		return
	}
	c.classifierMutex.Lock()
	c.classifier = classifier
	c.classifierMutex.Unlock()
}
