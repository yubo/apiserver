// +build linux darwin

//from github.com/yubo/gotty/rec
package rsh

import (
	"encoding/gob"
	"encoding/json"
	"io"
	"os"
	"sync/atomic"
	"time"

	"github.com/yubo/golib/staging/util/term"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

const (
	SampleTime = 100 * time.Millisecond // 10Hz
)

type CtlMsg struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type Player struct {
	FileName  string
	f         *os.File
	dec       *gob.Decoder
	d         RecData
	pending   bool
	speed     int64
	repeat    bool
	sync      bool
	fileStart int64
	window    struct {
		height uint16
		width  uint16
	}
	done     chan struct{}
	ctlMsgCh chan *CtlMsg

	pause    int64
	playTime int64
	start    int64
	offset   int64
	maxWait  int64
}

func NewPlayer(fileName string, speed int64, repeat bool, wait int64) (*Player, error) {
	var err error

	p := &Player{FileName: fileName,
		speed:    speed,
		repeat:   repeat,
		sync:     false,
		maxWait:  wait * 1000000,
		done:     make(chan struct{}),
		ctlMsgCh: make(chan *CtlMsg, 8),
	}

	if p.f, err = os.OpenFile(fileName, os.O_RDONLY, 0); err != nil {
		return nil, err
	}
	p.dec = gob.NewDecoder(p.f)

	p.run()

	return p, nil
}

func (p *Player) Read(d []byte) (n int, err error) {
	for {
		if !p.pending {
			if err = p.dec.Decode(&p.d); err != nil {
				if p.repeat && err == io.EOF {
					p.start = Nanotime()
					p.offset = 0
					atomic.StoreInt64(&p.playTime, 0)
					p.f.Seek(0, 0)
					p.dec = gob.NewDecoder(p.f)
					continue
				} else {
					return 0, err
				}
			}
			p.pending = true
		}

		switch p.d.Data[0] {
		case MsgResize:
			var args term.TerminalSize
			err = json.Unmarshal(p.d.Data[1:], &args)
			if err != nil {
				klog.Infof("Malformed remote command")
				continue
			}
			p.window.height = uint16(args.Height)
			p.window.width = uint16(args.Width)
			continue
		case MsgOutput, MsgInput:
			wait := p.d.Time - p.fileStart - p.offset -
				atomic.LoadInt64(&p.playTime)*p.speed
			if wait > p.maxWait {
				p.offset += wait - p.maxWait
			}
			for {
				wait = p.d.Time - p.fileStart - p.offset -
					atomic.LoadInt64(&p.playTime)*p.speed
				if wait <= 0 {
					break
				}

				// check chan before sleep
				select {
				case msg := <-p.ctlMsgCh:
					b := append([]byte{MsgCtl}, []byte(util.JsonStr(msg))...)
					klog.Infof("%s", string(b))
					n = copy(d, b[:])
					return
				default:
				}

				time.Sleep(time.Duration(MaxInt64(int64(SampleTime), wait)))
			}

			// synchronization time axis
			// expect wait == 0 when sending msg
			if p.sync {
				p.offset = p.d.Time - p.fileStart -
					atomic.LoadInt64(&p.playTime)*
						p.speed
			}
			n = copy(d, p.d.Data[:])
			p.pending = false

			return
		default:
			klog.Infof("unknow type(%d) context(%s)",
				p.d.Data[0], string(p.d.Data[1:]))
			continue
		}
	}
}

func MaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func (p *Player) run() {
	p.fileStart = p.d.Time
	p.start = Nanotime()
	go func() {
		tick := time.NewTicker(SampleTime)
		defer tick.Stop()
		var (
			lastTime = p.start
			now      = p.start
		)
		for {
			select {
			case <-p.done:
				return
			case t := <-tick.C:
				now = t.UnixNano()
				if atomic.LoadInt64(&p.pause)&0x01 == 0 {
					atomic.AddInt64(&p.playTime, now-lastTime)
				}
				lastTime = now
			}
		}
	}()
}

func (p *Player) Write(b []byte) (n int, err error) {
	n = len(b)

	if len(b) != 2 || b[0] != MsgInput {
		klog.Infof("player Write %d %s", len(b), string(b))
		return
	}

	switch b[1] {
	case 3, 4, 'q':
		return 0, io.EOF
	case ' ', 'p':
		if atomic.AddInt64(&p.pause, 1)&0x01 == 0 {
			p.ctlMsgCh <- &CtlMsg{Type: "unpause"}
		} else {
			p.ctlMsgCh <- &CtlMsg{Type: "pause"}
		}
	}

	return
}

func (p *Player) Close() error {
	close(p.done)
	return p.f.Close()
}
