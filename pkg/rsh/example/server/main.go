package main

import (
	"context"
	"flag"
	"net/http"

	restful "github.com/emicklei/go-restful"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/rsh"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

type ttyserver struct {
	rshd *rsh.Server
	cmd  []string
	env  []string
}

type ExecOption struct {
	Cmd []*string `param:"query" description:"cmd desc"`
	Foo *string   `param:"query" description:"foo desc"`
	Bar *int      `param:"query" description:"bar desc"`
}

func (p *ttyserver) WebService() *restful.WebService {
	ws := new(restful.WebService)
	ws.Path("/").Consumes(restful.MIME_JSON).Produces(restful.MIME_JSON) // you can specify this per route as well
	ws.Route(ws.GET("/exec").To(p.exec).Returns(200, "OK", ""))
	return ws
}

func (p *ttyserver) exec(req *restful.Request, resp *restful.Response) {
	in := &ExecOption{}
	if err := rest.ReadEntity(req, in, nil); err != nil {
		resp.WriteError(http.StatusInternalServerError, err)
		klog.V(3).Info(err)
		return
	}
	klog.V(3).Infof("recv payload %s", util.JsonStr(in))

	err := p.rshd.DataHandle(req, resp,
		func(data []byte) (cmd, env []string, err error) {
			klog.Infof("recv %s %s", string(data), util.JsonStr(in, true))
			return util.StringValueSlice(in.Cmd), p.env, nil
		})
	klog.V(3).Info(err)
}

func main() {
	klog.InitFlags(nil)
	flag.Set("v", "3")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	config := &rsh.RshConfig{
		BufferSize:  1024,
		PermitWrite: true,
	}

	rshd, err := rsh.NewServer(config, ctx)
	if err != nil {
		klog.Fatal(err)
		return
	}

	server := &ttyserver{
		rshd: rshd,
		cmd:  []string{"bash"},
		env:  []string{"FOO=1", "FOO=2"},
	}

	restful.DefaultContainer.Add(server.WebService())
	klog.Infof("start listening on localhost:18080")
	klog.Fatal(http.ListenAndServe(":18080", nil))
}
