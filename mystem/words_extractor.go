package mystem

import (
	"bufio"
	"errors"
	"strings"
)

type WordsExtractor struct {
	binPath string
}

func NewWordsExtractor(binPath string) *WordsExtractor {
	return &WordsExtractor{binPath: binPath}
}

func (ke *WordsExtractor) ExtractWords(text string) (
	[]string, error) {

	if text == "" {
		return nil, nil
	}

	res, err := ke.runMystem(strings.NewReader(text))
	if err != nil {
		return nil, errors.New("failed to run mystem: " + err.Error())
	}

	scanner := bufio.NewScanner(res)

	kwsMap := map[string]struct{}{}

	for scanner.Scan() {
		line := scanner.Text()
		for _, kw := range strings.Split(line, "|") {
			kw = strings.ToLower(strings.TrimRight(kw, "?"))
			if !isStopWord(kw) {
				kwsMap[kw] = struct{}{}
			}
		}
	}

	var kws []string
	for kw := range kwsMap {
		kws = append(kws, kw)
	}

	return kws, nil
}
