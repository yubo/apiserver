package cmdcli

import (
	"context"
	"io"
	"time"

	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/golib/scheme"
)

// host: 127.0.0.1:8080
func NewRequest(host string, opts ...RequestOption) (*Request, error) {
	client, err := rest.RESTClientFor(&rest.Config{
		Host:          host,
		ContentConfig: rest.ContentConfig{NegotiatedSerializer: scheme.Codecs},
	})

	if err != nil {
		return nil, err
	}
	return NewRequestWithClient(client, opts...), nil
}

func NewRequestWithClient(client *rest.RESTClient, opts ...RequestOption) *Request {
	o := &Request{
		client: client,
	}

	for _, opt := range opts {
		opt.apply(&o.RequestOptions)
	}
	return o
}

func (p *Request) Pager(stdout io.Writer, disablePage bool) *Pager {
	pager, err := NewPager(p, stdout, disablePage)
	if err != nil {
		panic(err)
	}
	return pager
}

type Request struct {
	client *rest.RESTClient
	RequestOptions
}

// ("GET", "https://example.com/api/v{version}/{model}/{subject}?a=1&b=2", {"subject":"abc", "model": "instance", "version": 1}, nil)
func (p *Request) Do(ctx context.Context) error {
	req := p.client.Verb(p.method)

	if p.prefix != "" {
		req = req.Prefix(p.prefix)
	}

	if p.input != nil {
		req = req.VersionedParams(p.input, scheme.ParameterCodec)
	}

	if p.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.timeout)
		defer cancel()
	}

	if w, ok := p.output.(io.Writer); ok {
		b, err := req.Do(ctx).Raw()
		if err != nil {
			return err
		}

		if _, err := w.Write(b); err != nil {
			return err
		}
		w.Write([]byte("\n"))
		return nil
	}

	if err := req.Do(ctx).Into(p.output); err != nil {
		return err
	}

	for _, fn := range p.cb {
		if fn != nil {
			fn(p.output)
		}
	}

	return nil
}

type RequestOptions struct {
	method  string
	prefix  string
	timeout time.Duration // second
	input   interface{}
	output  interface{}
	cb      []func(interface{})
}

type RequestOption interface {
	apply(*RequestOptions)
}

type funcRequestOption struct {
	f func(*RequestOptions)
}

func (p *funcRequestOption) apply(opt *RequestOptions) {
	p.f(opt)
}

func newFuncRequestOption(f func(*RequestOptions)) *funcRequestOption {
	return &funcRequestOption{
		f: f,
	}
}

func WithMethod(method string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.method = method
	})
}
func WithPrifix(prefix string) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.prefix = prefix
	})
}
func WithInput(input interface{}) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.input = input
	})
}
func WithOutput(output interface{}) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.output = output
	})
}
func WithCallback(cb ...func(interface{})) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.cb = cb
	})
}
func WithTimeout(timeout time.Duration) RequestOption {
	return newFuncRequestOption(func(o *RequestOptions) {
		o.timeout = timeout
	})
}
