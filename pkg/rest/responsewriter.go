package rest

import (
	"fmt"
	"io"
	"net/http"

	"github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"k8s.io/klog/v2"
)

type RespStatus struct {
	Code int    `json:"code" description:"status code"`
	Err  string `json:"err" description:"error msg"`
}

type RespTotal struct {
	Total int64 `json:"total" description:"total number"`
}

type RespID struct {
	ID int64 `json:"id" description:"id"`
}

func NewRespStatus(err error) (resp RespStatus) {
	status := responsewriters.ErrorToAPIStatus(err)
	resp.Err = status.Message
	resp.Code = http.StatusBadRequest

	return
}

func NewRespID(id int64) RespID {
	return RespID{ID: id}
}

func NewRespTotal(total int64) RespTotal {
	return RespTotal{Total: total}
}

func RespWrite(resp *restful.Response, req *http.Request, data interface{}, err error) {
	if err != nil {
		code := responsewriters.Error(err, resp, req)
		klog.V(3).Infof("response %d %s", code, err.Error())
		return
	}

	if _, ok := data.(NoneParam); ok {
		return
	}

	if b, ok := data.([]byte); ok {
		resp.Write(b)
		return
	}

	resp.WriteEntity(data)

}

// wrapper data and error
// TODO: use http.ResponseWriter instead of restful.Response
func RespWriteErrInBody(resp *restful.Response, data interface{}, err error) {
	var eMsg string
	status := responsewriters.ErrorToAPIStatus(err)
	code := status.Code

	if err != nil {
		eMsg = err.Error()

		if klog.V(3).Enabled() {
			klog.ErrorDepth(1, fmt.Sprintf("httpReturn %d %s", code, eMsg))
		}
	}

	resp.WriteEntity(map[string]interface{}{
		"data": data,
		"err":  eMsg,
		"code": code,
	})
}

//  Deprecated
func HttpWriteData(resp *restful.Response, data interface{}, err error, tx ...Tx) {
	var eMsg string
	status := responsewriters.ErrorToAPIStatus(err)
	code := int(status.Code)

	if len(tx) > 0 && tx != nil {
		txClose(tx[0], err)
	}

	if err != nil {
		eMsg = err.Error()

		if klog.V(3).Enabled() {
			klog.ErrorDepth(1, fmt.Sprintf("httpReturn %d %s", code, eMsg))
		}
	}

	resp.WriteEntity(map[string]interface{}{
		"data": data,
		"err":  eMsg,
		"code": code,
	})
}

func HttpWriteList(resp *restful.Response, total int64, list interface{}, err error) {
	var eMsg string
	status := responsewriters.ErrorToAPIStatus(err)
	code := int(status.Code)

	if err != nil {
		eMsg = err.Error()
	} else if list == nil {
		list = []string{}
	}

	resp.WriteEntity(map[string]interface{}{
		"data": map[string]interface{}{
			"total": total,
			"list":  list,
		},
		"err":  eMsg,
		"code": code,
	})
}

func HttpRedirect(w http.ResponseWriter, url string) {
	w.Header().Add("location", url)
	w.WriteHeader(http.StatusFound)
}

func HttpRedirectErr(resp *restful.Response, url string, err error) {
	if err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
		return
	}
	resp.ResponseWriter.Header().Add("location", url)
	resp.ResponseWriter.WriteHeader(http.StatusFound)
}

func HttpWriteEntity(resp *restful.Response, in interface{}, err error) {
	if err != nil {
		HttpWriteErr(resp, err)
		return
	}

	if s, ok := in.(string); ok {
		resp.Write([]byte(s))
		return
	}

	resp.WriteEntity(in)
}

func HttpWrite(resp *restful.Response, data []byte, err error) {
	if err != nil {
		HttpWriteErr(resp, err)
		return
	}

	resp.Write(data)
}

func HttpWriteErr(resp *restful.Response, err error) {
	if err == nil {
		return
	}

	status := responsewriters.ErrorToAPIStatus(err)
	code := int(status.Code)
	resp.WriteError(code, err)
}

func HttpRespPrint(out io.Writer, resp *http.Response, body []byte) {
	if out == nil || resp == nil {
		return
	}

	fmt.Fprintf(out, "[resp]\ncode: %d\n", resp.StatusCode)
	fmt.Fprintf(out, "header:\n")

	for k, v := range resp.Header {
		for _, v1 := range v {
			fmt.Fprintf(out, "  %s: %s\n", k, v1)
		}
	}

	if len(body) > 0 {
		fmt.Fprintf(out, "body:\n%s\n", string(body))
	}
}

type Tx interface {
	Tx() bool
	Commit() error
	Rollback() error
}

func txClose(tx Tx, err error) error {
	if tx == nil || !tx.Tx() {
		return nil
	}

	if err == nil {
		return tx.Commit()
	}
	return tx.Rollback()
}
