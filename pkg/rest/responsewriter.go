package rest

import (
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"k8s.io/klog/v2"
)

func DefaultRespWrite(resp *restful.Response, req *http.Request, data interface{}, err error) {
	if err != nil {
		code := responsewriters.Error(err, resp, req)
		klog.V(3).Infof("response %d %s", code, err.Error())
		return
	}

	if _, ok := data.(NonParam); ok {
		resp.WriteHeader(http.StatusOK)
		return
	}

	if b, ok := data.([]byte); ok {
		resp.Write(b)
		return
	}

	resp.WriteEntity(data)
}

// wrapper data and error
func RespWriteErrInBody(resp *restful.Response, req *http.Request, data interface{}, err error) {
	v := map[string]interface{}{"data": data}
	code := http.StatusOK

	if err != nil {
		v["err"] = err.Error()
		code = int(responsewriters.ErrorToAPIStatus(err).Code)
	}

	v["code"] = code

	resp.WriteEntity(v)
}
