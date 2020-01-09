package bayesian

import (
	"encoding/gob"
	"errors"
	"fmt"
	"math"
	"os"
	"sync/atomic"

	"github.com/dimuls/classifier-v2/entity"
)

type Tokenizer interface {
	Tokenize(text string) (map[string]int, error)
}

type Classifier struct {
	tokenizer Tokenizer
	filePath  string

	data map[string]map[string]int

	training int32
}

func NewClassifier(t Tokenizer, filePath string) (*Classifier, error) {
	c := &Classifier{
		tokenizer: t,
		filePath:  filePath,
		data:      map[string]map[string]int{},
	}

	err := c.load()
	if err != nil {
		return nil, fmt.Errorf("load classifier: %w", err)
	}

	return c, nil
}

func (s *Classifier) load() error {
	f, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open file: %w", err)
	}

	err = gob.NewDecoder(f).Decode(&s.data)
	if err != nil {
		return fmt.Errorf("gob decode: %w", err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}

func (s *Classifier) save() error {
	f, err := os.Create(s.filePath)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	err = gob.NewEncoder(f).Encode(s.data)
	if err != nil {
		return fmt.Errorf("gob encode: %w", err)
	}

	err = f.Close()
	if err != nil {
		return fmt.Errorf("close file: %w", err)
	}

	return nil
}

func (c *Classifier) Train(docs []entity.Document) error {
	atomic.StoreInt32(&c.training, 1)
	defer atomic.StoreInt32(&c.training, 0)

	for _, d := range docs {

		wordsCounts, err := c.tokenizer.Tokenize(d.Text)
		if err != nil {
			return fmt.Errorf("tokenize text: %w", err)
		}

		if classWordsCounts, exists := c.data[d.Class]; exists {
			for word, count := range wordsCounts {
				classWordsCounts[word] += count
			}
		} else {
			c.data[d.Class] = wordsCounts
		}
	}

	err := c.save()
	if err != nil {
		return fmt.Errorf("save classifier: %w", err)
	}

	return nil
}

func (s *Classifier) RemoveFile() error {
	return os.Remove(s.filePath)
}

func (c *Classifier) Training() bool {
	return atomic.LoadInt32(&c.training) == 1
}

func keys(m map[string]int) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	return
}

// classesTotalWordsCounts returns `class -> total words count` map
func (s *Classifier) classesTotalWordsCounts() map[string]int {

	result := map[string]int{}

	for class, wordsCount := range s.data {
		for _, count := range wordsCount {
			result[class] += count
		}
	}

	return result
}

// classWordsCounts returns `word -> count` map for given class and words
func (s *Classifier) classWordsCounts(class string,
	words []string) map[string]int {

	result := map[string]int{}

	wordsCount, exists := s.data[class]
	if !exists {
		for _, w := range words {
			result[w] = 0
		}
		return result
	}

	for _, w := range words {
		result[w] = wordsCount[w]
	}

	return result
}

const defaultWordProb float64 = 1e-11

func (c *Classifier) Classify(text string) (string, error) {
	if c.Training() {
		return "", errors.New("classifier is training")
	}

	wordsCounts, err := c.tokenizer.Tokenize(text)
	if err != nil {
		return "", fmt.Errorf("tokenize text: %w", err)
	}

	classesTotalWordsCounts := c.classesTotalWordsCounts()

	words := keys(wordsCounts)
	scores := map[string]float64{}

	for class, classTotalWordsCount := range classesTotalWordsCounts {
		classWordsCounts := c.classWordsCounts(class, words)

		var score float64

		for _, word := range words {
			wordProb := float64(classWordsCounts[word]) /
				float64(classTotalWordsCount)
			if wordProb == 0 {
				wordProb = defaultWordProb
			}
			score += math.Log(wordProb)
		}

		scores[class] = score
	}

	maxScore := -math.MaxFloat64
	var maxClass string

	for class, score := range scores {
		if score > maxScore {
			maxScore = score
			maxClass = class
		}
	}

	return maxClass, nil
}
