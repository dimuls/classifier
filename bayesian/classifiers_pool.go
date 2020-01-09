package bayesian

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	classifierFileExtension = "bc"
	classifierFileSuffix    = "." + classifierFileExtension
)

type ClassifiersPool struct {
	tokenizer   Tokenizer
	dataPath    string
	classifiers map[string]*Classifier
	mx          sync.Mutex
}

func NewClassifiersPool(t Tokenizer, dataPath string) (*ClassifiersPool, error) {
	cp := &ClassifiersPool{
		tokenizer:   t,
		dataPath:    dataPath,
		classifiers: map[string]*Classifier{},
	}

	err := filepath.Walk(dataPath, cp.walk)
	if err != nil {
		return nil, fmt.Errorf("walk data path: %w", err)
	}

	return cp, nil
}

func (cp *ClassifiersPool) Classifier(classifierID string, create bool) (
	*Classifier, bool) {

	cp.mx.Lock()
	defer cp.mx.Unlock()

	c, exist := cp.classifiers[classifierID]
	if !exist && create {

		var err error

		c, err = NewClassifier(cp.tokenizer, path.Join(cp.dataPath,
			classifierID+classifierFileSuffix))
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"classifier_id": classifierID,
				"create":        create,
				"subsystem":     "bayesian_classifiers_pool",
			}).WithError(err).Fatal("expected no error")
		}

		cp.classifiers[classifierID] = c
	}

	return c, exist
}

func (cp *ClassifiersPool) RemoveClassifier(classifierID string) error {
	c, exists := cp.Classifier(classifierID, false)
	if !exists {
		return errors.New("classifier not exists")
	}

	err := c.RemoveFile()
	if err != nil {
		return fmt.Errorf("remove classifier file: %w", err)
	}

	cp.mx.Lock()
	delete(cp.classifiers, classifierID)
	cp.mx.Unlock()

	return nil
}

func (cp *ClassifiersPool) walk(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if info.IsDir() {
		return nil
	}
	if !strings.HasSuffix(info.Name(), classifierFileSuffix) {
		return nil
	}

	c, err := NewClassifier(cp.tokenizer, path)
	if err != nil {
		return fmt.Errorf("new classifier: %w", err)
	}

	cID := strings.TrimSuffix(info.Name(), classifierFileSuffix)

	cp.classifiers[cID] = c

	return nil
}
