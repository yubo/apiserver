package rsh

import (
	"io"

	"k8s.io/klog/v2"
)

type FdxFilterFunction func([]byte) ([]byte, error)

// Full Duplex Pipe
type Fdx struct {
	bufferSize int
	conn       io.ReadWriter
	upstream   io.ReadWriter
	rxFilter   []FdxFilterFunction
	txFilter   []FdxFilterFunction
}

func NewFdx(conn, upstream io.ReadWriter, bsize int) (fdx *Fdx, err error) {
	return &Fdx{
		conn:       conn,
		upstream:   upstream,
		bufferSize: bsize,
	}, nil
}

// conn <- upstream
func (p *Fdx) RxFilter(f FdxFilterFunction) *Fdx {
	p.rxFilter = append(p.rxFilter, f)
	return p
}

// conn -> upstream
func (p *Fdx) TxFilter(f FdxFilterFunction) *Fdx {
	p.txFilter = append(p.txFilter, f)
	return p
}

func (p *Fdx) Run() (err error) {
	errs := make(chan error, 2)
	defer func() {
		klog.Infof("tty return %v", err)
	}()

	// conn -> upstream
	go func() {
		buff := make([]byte, p.bufferSize)
		_, err := fdxCopyBuffer(p.upstream, p.conn, buff, p.txFilter)
		errs <- err
	}()

	// conn <- upstream
	go func() {
		buff := make([]byte, p.bufferSize)
		_, err := fdxCopyBuffer(p.conn, p.upstream, buff, p.rxFilter)
		errs <- err
	}()

	err = <-errs
	return
}

func fdxCopyBuffer(dst io.Writer, src io.Reader, buf []byte, filter []FdxFilterFunction) (written int64, err error) {
	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}

	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			b := buf[:nr]
			for _, f := range filter {
				b, err = f(b)
				if err != nil {
					return written, err
				}
			}
			nw, ew := write(dst, b)
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}

func write(w io.Writer, data []byte) (int, error) {
	len := len(data)

	for i := 0; i < len; {
		if n, err := w.Write(data[i:]); err != nil {
			return i, err
		} else {
			i += n
		}
	}
	return len, nil
}
