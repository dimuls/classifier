package mystem

import (
	"bytes"
	"io"
	"os/exec"
)

func (ke *WordsExtractor) runMystem(stdin io.Reader) (io.Reader, error) {

	stdout := bytes.NewBuffer(nil)

	cmd := exec.Command(ke.binPath, "-n", "-l")
	cmd.Stdin = stdin
	cmd.Stdout = stdout

	err := cmd.Run()

	return stdout, err
}
