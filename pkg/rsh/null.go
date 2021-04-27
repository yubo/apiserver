package rsh

import "errors"

var ErrClosed = errors.New("io: read/write on closed")

type Null struct {
	done chan struct{}
}

func NewNull() *Null {
	return &Null{done: make(chan struct{})}
}

func (p *Null) Write(b []byte) (int, error) {
	select {
	case <-p.done:
		return 0, ErrClosed
	default:
	}

	return len(b), nil
}

func (p *Null) Read(b []byte) (int, error) {
	select {
	case <-p.done:
		return 0, ErrClosed
	default:
	}

	if len(b) > 0 {
		<-p.done
	}

	return 0, nil
}

func (p *Null) Close() error {
	select {
	case <-p.done:
		return ErrClosed
	default:
	}

	close(p.done)
	return nil
}
