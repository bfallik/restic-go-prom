package restic

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os/exec"
)

type Cmd struct {
	*exec.Cmd

	OutBuf bytes.Buffer
	ErrBuf bytes.Buffer
}

type CmdOptRepo struct {
	Repo string
}

func (c CmdOptRepo) GetArgs() []string {
	return []string{"--repo", c.Repo}
}

type CmdOptCommandBackup struct{}

func (CmdOptCommandBackup) GetArgs() []string { return []string{"backup"} }

type CmdOptCommandStats struct{}

func (CmdOptCommandStats) GetArgs() []string { return []string{"stats"} }

type CmdOptCommandVersion struct{}

func (CmdOptCommandVersion) GetArgs() []string { return []string{"version"} }

type CmdOptFlags []string

func (c CmdOptFlags) GetArgs() []string { return c }

type cmdopts interface {
	GetArgs() []string
}

func NewCmd(opts ...cmdopts) (*Cmd, error) {
	args := []string{}
	for _, opt := range opts {
		args = append(args, opt.GetArgs()...)
	}

	cmd := Cmd{
		Cmd: exec.Command("restic", args...),
	}
	if cmd.Err != nil {
		return nil, cmd.Err
	}

	cmd.Stdout = &cmd.OutBuf
	cmd.Stderr = &cmd.ErrBuf

	return &cmd, nil
}

func (cmd *Cmd) JSONLines() ([]json.RawMessage, error) {
	res := []json.RawMessage{}
	scanner := bufio.NewScanner(&cmd.OutBuf)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		raw := json.RawMessage{}
		if err := json.Unmarshal(scanner.Bytes(), &raw); err != nil {
			return nil, err
		}
		res = append(res, raw)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return res, nil
}
