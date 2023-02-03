package rest

import (
	"io/ioutil"
	"reflect"
	goruntime "runtime"

	"github.com/emicklei/go-restful/v3"
	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/handlers/negotiation"
	"github.com/yubo/apiserver/pkg/request"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/util/errors"
)

type requestType int

const (
	paramType requestType = iota
	bodyType
)

type routeHandle struct {
	serializer     runtime.NegotiatedSerializer
	parameterCodec request.ParameterCodec
	respWriter     RespWriter
	handle         interface{}
	ht             reflect.Type
	hv             reflect.Value
	in             []requestType
	param          reflect.Type // request param - query, path, header
	body           reflect.Type // request body
	out            reflect.Type
}

func NewRouteHandle(
	handle interface{},
	parameterCodec request.ParameterCodec,
	serializer runtime.NegotiatedSerializer,
	respWriter RespWriter,
) (*routeHandle, error) {

	ret := &routeHandle{
		handle:         handle,
		parameterCodec: parameterCodec,
		serializer:     serializer,
		respWriter:     respWriter,
	}

	if err := ret.init(); err != nil {
		return nil, err
	}

	return ret, nil
}

func (p *routeHandle) init() error {
	p.hv = reflect.ValueOf(p.handle)
	p.ht = p.hv.Type()

	// validate
	if err := p.validateHandle(); err != nil {
		return errors.Wrapf(err, "validate handle function %s %s",
			goruntime.FuncForPC(p.hv.Pointer()).Name(), p.ht.String())
	}

	if err := p.initHandleIO(); err != nil {
		return errors.Wrapf(err, "init handleIO")
	}

	return nil
}

func (p *routeHandle) validateHandle() error {
	rt := p.ht
	numIn := rt.NumIn()
	numOut := rt.NumOut()

	if numIn < 2 || numIn > 4 {
		return errors.Errorf("handle.NumIn() %d expected [2,4]", numIn)
	}

	if numOut > 2 {
		return errors.Errorf("handle.NumOut() %d expected [0, 2]", numOut)
	}

	if arg := rt.In(0).String(); arg != "http.ResponseWriter" {
		return errors.Errorf("expected func(*http.Request, http.ResponseWriter, ...)")
	}

	if arg := rt.In(1).String(); arg != "*http.Request" {
		return errors.Errorf("expected func(*http.Request, http.ResponseWriter, ...)")
	}

	return nil
}

// handle(req *restful.Request, resp *restful.Response)
// handle(req *restful.Request, resp *restful.Response, param *struct{})
// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *slice)
// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *map)
// handle(req *restful.Request, resp *restful.Response, param *struct{}, body *struct)
// handle(...) error
// handle(...) (out *struct{}, err error)

// func (f HandlerFunc) ServeHTTP(w ResponseWriter, r *Request) {
// handle(w ResponseWriter, r *Request, param *struct{}, body *struct{})
func (p *routeHandle) initHandleIO() error {

	// in
	for i := 2; i < p.ht.NumIn(); i++ {
		rt := p.ht.In(i)

		if rt.Kind() != reflect.Ptr {
			return errors.New("payload must be a ptr")
		}
		rt = rt.Elem()

		if p.isParam(rt) {
			if err := validateParamType(rt); err != nil {
				return errors.Wrap(err, "request param type invlid")
			}
			if p.param != nil {
				return errors.New("duplicate request param field")
			}
			p.param = rt
			p.in = append(p.in, paramType)
			continue
		}

		// body
		if err := validateBodyType(rt); err != nil {
			return errors.Wrap(err, "get request body type")
		}
		if p.body != nil {
			return errors.New("duplicate request body field")
		}
		p.body = rt
		p.in = append(p.in, bodyType)
	}

	// out
	if p.ht.NumOut() == 2 {
		rt, err := getResponseType(p.ht.Out(0))
		if err != nil {
			return errors.Wrap(err, "get response type")
		}
		p.out = rt
	}

	return nil
}

func (p *routeHandle) isParam(rt reflect.Type) bool {
	return p.parameterCodec.ValidateParamType(rt) == nil
}

func (p *routeHandle) Handler() func(req *restful.Request, resp *restful.Response) {
	return func(req *restful.Request, resp *restful.Response) {
		param := newInterface(p.param)
		body := newInterface(p.body)

		if err := p.readEntity(req, param, body); err != nil {
			p.respWriter.RespWrite(resp, req.Request, nil, err, p.serializer)
			return
		}

		// audit
		ae := request.AuditEventFrom(req.Request.Context())
		audit.LogRequestObject(ae, body, "")

		// call handle
		in := []reflect.Value{
			reflect.ValueOf(resp.ResponseWriter),
			reflect.ValueOf(req.Request),
		}
		for _, v := range p.in {
			switch v {
			case paramType:
				in = append(in, reflect.ValueOf(param))
			case bodyType:
				in = append(in, reflect.ValueOf(body))
			}
		}
		ret := p.hv.Call(in)

		var err error
		switch len(ret) {
		case 1:
			err = toError(ret[0])
			p.respWriter.RespWrite(resp, req.Request, nil, err, p.serializer)
		case 2:
			err = toError(ret[1])
			p.respWriter.RespWrite(resp, req.Request, toInterface(ret[0]), err, p.serializer)
		}
		if err != nil {
			req.SetAttribute("error", err)
		}
	}
}

// dst: must be ptr
func (p *routeHandle) readEntity(req *restful.Request, param, body interface{}) error {
	return readEntity(req, param, body, p.parameterCodec, p.serializer)
}

func readEntity(req *restful.Request, param, body interface{}, codec request.ParameterCodec, serializer runtime.NegotiatedSerializer) error {
	if param != nil {
		//ctx := request.WithParam(req.Request.Context(), param)
		//req.Request = req.Request.WithContext(ctx)

		if err := codec.DecodeParameters(&request.Parameters{
			Header: req.Request.Header,
			Path:   req.PathParameters(),
			Query:  req.Request.URL.Query(),
		}, param); err != nil {
			return nil
		}
		if v, ok := param.(Validator); ok {
			if err := v.Validate(); err != nil {
				return err
			}
		}

	}

	if body != nil {
		//ctx = request.WithBody(req.Request.Context(), body)
		//req.Request = req.Request.WithContext(ctx)

		// body
		if body != nil {
			s, err := negotiation.NegotiateInputSerializer(req.Request, false, serializer)
			if err != nil {
				return err
			}

			buff, err := ioutil.ReadAll(req.Request.Body)
			if err != nil {
				return err
			}

			if _, err := s.Serializer.Decode(buff, body); err != nil {
				return err
			}
			//if err := req.ReadEntity(body); err != nil {
			//	klog.V(5).Infof("restful.ReadEntity() error %s", err)
			//	return err
			//}
		}
		if v, ok := body.(Validator); ok {
			if err := v.Validate(); err != nil {
				return err
			}
		}
	}

	return nil

}

func validateParamType(rt reflect.Type) error {
	switch rt.Kind() {
	case reflect.Struct:
	default:
		return errors.Errorf("param just support ptr to struc")
	}

	return nil
}

func validateBodyType(rt reflect.Type) error {
	switch rt.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Map:
	default:
		return errors.Errorf("just support ptr to struct|slice|map")
	}

	return nil
}

func getResponseType(rt reflect.Type) (reflect.Type, error) {
	if rt.Kind() == reflect.Ptr {
		rt = rt.Elem()
	}
	return rt, nil
}
