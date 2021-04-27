// +build linux darwin

//from github.com/yubo/gotty/rec
package rsh

import (
	"encoding/gob"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

type RecData struct {
	Time int64
	Data []byte
}

type Recorder struct {
	FileName string
	f        *os.File
	enc      *gob.Encoder
}

func EnvsToKv(envs []string) map[string]string {
	data := map[string]string{}
	for _, env := range envs {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) == 2 {
			data[pair[0]] = pair[1]
		}
	}
	return data
}

// TERM SHELL COMMAND
func NewRecorder(fileName string) (r *Recorder, err error) {

	r = &Recorder{FileName: fileName}
	if r.f, err = os.Create(fileName); err != nil {
		return nil, err
	}
	r.enc = gob.NewEncoder(r.f)

	return r, nil
}

func (r *Recorder) Read(d []byte) (n int, err error) {
	return 0, io.EOF
}

func (r *Recorder) Write(d []byte) (n int, err error) {
	if err := r.enc.Encode(RecData{Time: Nanotime(), Data: d}); err != nil {
		return 0, err
	}
	return len(d), nil
}

func (r *Recorder) Close() error {
	return r.f.Close()
}

func RecordRun(cmd *exec.Cmd, fileName string, input bool) error {
	cli := &Client{
		IOStreams: IOStreams{
			In:     os.Stdin,
			Out:    os.Stdout,
			ErrOut: nil,
		},
		Stdin: true,
		TTY:   true,
	}

	t := cli.Setup()
	cli.SizeQueue = t.MonitorSize(t.GetSize()).Ch()

	return t.Safe(func() error {
		cmd.Env = append(os.Environ(), "REC=[REC]")

		pty, err := NewPty()
		defer pty.Close()

		cmd.Stdout = pty.Tty
		cmd.Stdin = pty.Tty
		cmd.Stderr = pty.Tty
		if cmd.SysProcAttr == nil {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		cmd.SysProcAttr.Setsid = true
		cmd.SysProcAttr.Setctty = true
		cmd.SysProcAttr.Ctty = int(pty.Tty.Fd())

		fdx, err := NewFdx(&cli.IOStreams, pty.Pty, RshBuffSize)
		if err != nil {
			return err
		}

		recorder, err := NewRecorder(fileName)
		if err != nil {
			return err
		}
		defer recorder.Close()

		// tty <- cmd
		fdx.RxFilter(func(b []byte) ([]byte, error) {
			recorder.Write(append([]byte{MsgOutput}, b...))
			return b, nil
		})

		// tty -> cmd
		fdx.TxFilter(func(b []byte) ([]byte, error) {
			if input {
				recorder.Write(append([]byte{MsgInput}, b...))
			}
			return b, nil
		})

		go func() {
			for {
				ws, ok := <-cli.SizeQueue
				if !ok {
					return
				}
				pty.Resize(ws.Width, ws.Height)
			}
		}()

		go fdx.Run()

		return cmd.Run()
	})

}
