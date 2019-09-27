package mystem

import (
	"bytes"
	"io"
	"os/exec"
	"syscall"
)

func (ke *WordsExtractor) runMystem(stdin io.Reader) (io.Reader, error) {

	stdout := bytes.NewBuffer(nil)

	cmd := exec.Command(ke.binPath, "-n", "-l")
	cmd.Stdin = stdin
	cmd.Stdout = stdout

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Run()

	return stdout, err
}
