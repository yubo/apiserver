// +build linux darwin

// convert rsh rec format to github.com/asciinema/asciinema format
package convert

import (
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/yubo/apiserver/pkg/rsh"
	"github.com/yubo/golib/staging/util/term"
)

type Duration float64

func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`%.6f`, d)), nil
}

type Asciicast struct {
	Version  int      `json:"version"`
	Width    int      `json:"width"`
	Height   int      `json:"height"`
	Duration Duration `json:"duration"`
	Command  string   `json:"command"`
	Title    string   `json:"title"`
	Env      *Env     `json:"env"`
	Stdout   []Frame  `json:"stdout"`
}

type Env struct {
	Term  string `json:"TERM"`
	Shell string `json:"SHELL"`
}

type Stream struct {
	Frames        []Frame
	maxWait       int64
	lastWriteTime int64
	elapsedTime   int64
	init          bool
}

func (s *Stream) Write(time int64, p []byte) (int, error) {
	if !s.init {
		s.lastWriteTime = time
		s.init = true
	}
	frame := Frame{}
	frame.Delay = s.incrementElapsedTime(time)
	frame.Data = make([]byte, len(p))
	copy(frame.Data, p)
	s.Frames = append(s.Frames, frame)

	return len(p), nil
}

func (s *Stream) Close() {
	s.incrementElapsedTime(s.lastWriteTime)
}

func nano2sec(d int64) float64 {
	sec := d / 1000000000
	nsec := d % 1000000000
	return float64(sec) + float64(nsec)*1e-9
}

func (s *Stream) incrementElapsedTime(time int64) float64 {
	d := time - s.lastWriteTime

	if s.maxWait > 0 && d > s.maxWait {
		d = s.maxWait
	}

	s.elapsedTime += d
	s.lastWriteTime = time

	return nano2sec(d)
}

func Save(asciicast *Asciicast, path string) error {
	bytes, err := json.MarshalIndent(asciicast, "", "  ")
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, bytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func Convert(src, dst string, wait int64) error {
	var buf rsh.RecData

	fp, err := os.OpenFile(src, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer fp.Close()

	dec := gob.NewDecoder(fp)
	s := &Stream{maxWait: wait * 1000000000}

	asciicast := &Asciicast{Version: 1, Env: &Env{}}
	if err = dec.Decode(&buf); err != nil {
		if err == io.EOF {
			return errors.New("empty file")
		}
	}

	for {
		switch buf.Data[0] {
		case rsh.MsgResize:
			var args term.TerminalSize
			err = json.Unmarshal(buf.Data[1:], &args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Malformed remote command")
			} else {
				asciicast.Height = int(args.Height)
				asciicast.Width = int(args.Width)
			}
		case rsh.MsgOutput:
			s.Write(buf.Time, buf.Data[1:])
		default:
			fmt.Fprintf(os.Stderr, "unknow type(%d) context(%s)",
				buf.Data[0], string(buf.Data[1:]))
		}

		if err = dec.Decode(&buf); err != nil {
			if err == io.EOF {
				break
			}
		}
	}
	asciicast.Stdout = s.Frames
	asciicast.Duration = Duration(nano2sec(s.elapsedTime))
	return Save(asciicast, dst)
}
