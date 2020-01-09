package mystem

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os/exec"
	"strings"
	"syscall"
)

type Tokenizer struct {
	binPath string
}

func NewTokenizer(binPath string) *Tokenizer {
	return &Tokenizer{binPath: binPath}
}

func (t *Tokenizer) Tokenize(text string) (map[string]int, error) {

	if text == "" {
		return nil, nil
	}

	res, err := t.mystem(strings.NewReader(text))
	if err != nil {
		return nil, errors.New("failed to run mystem: " + err.Error())
	}

	scanner := bufio.NewScanner(res)

	tokens := map[string]int{}

	for scanner.Scan() {
		line := scanner.Text()
		for _, kw := range strings.Split(line, "|") {
			kw = strings.ToLower(strings.TrimRight(kw, "?"))
			if !isStopWord(kw) {
				tokens[kw]++
			}
		}
	}

	return tokens, nil
}

func (t *Tokenizer) mystem(stdin io.Reader) (io.Reader, error) {

	stdout := bytes.NewBuffer(nil)

	cmd := exec.Command(t.binPath, "-n", "-l")
	cmd.Stdin = stdin
	cmd.Stdout = stdout

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Run()

	return stdout, err
}