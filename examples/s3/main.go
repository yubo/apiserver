package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/yubo/apiserver/pkg/options"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/apiserver/pkg/rest"
	"github.com/yubo/apiserver/pkg/s3"
	server "github.com/yubo/apiserver/pkg/server/module"
	"github.com/yubo/golib/cli"
	"github.com/yubo/golib/proc"

	_ "github.com/yubo/apiserver/pkg/s3/register"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

const (
	moduleName = "client.s3.examples"
)

type module struct {
	name string
	s3   s3.S3Client
}

var (
	_module = &module{name: moduleName}
	hookOps = []proc.HookOps{{
		Hook:     _module.start,
		Owner:    moduleName,
		HookNum:  proc.ACTION_START,
		Priority: proc.PRI_MODULE,
	}}
)

func main() {
	command := proc.NewRootCmd(server.WithoutTLS(), proc.WithHooks(hookOps...))
	code := cli.Run(command)
	os.Exit(code)
}

func (p *module) start(ctx context.Context) error {
	http, ok := options.APIServerFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get http server from the context")
	}

	s3, ok := options.S3ClientFrom(ctx)
	if !ok {
		return fmt.Errorf("unable to get s3 client from the context")
	}
	p.s3 = s3

	p.installWs(http)
	return nil
}

func (p *module) installWs(c rest.GoRestfulContainer) {
	c.HandlePrefix("/s3/", p)
}

func (p *module) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	var err error
	switch req.Method {
	case "GET":
		err = p.getfile(w, req)
	case "POST":
		err = p.putfile(w, req)
	case "DELETE":
		err = p.deletefile(w, req)
	default:
		err = fmt.Errorf("unsupport method %s", req.Method)
	}
	if err != nil {
		responsewriters.Error(err, w, req)
	}
}

func (p *module) putfile(w http.ResponseWriter, req *http.Request) error {
	if err := req.ParseMultipartForm(32 << 20); err != nil {
		return err
	}

	fd, fi, err := req.FormFile("uploadFile")
	if err != nil {
		return err
	}
	defer fd.Close()

	objectName := path.Join(getObjectPath(req), path.Base(fi.Filename))

	// TODO: contextType
	if err := p.s3.Put(req.Context(), objectName, "", fd, fi.Size); err != nil {
		return err
	}

	responsewriters.WriteRawJSON(200, objectName, w)
	return nil
}

// proxy -> s3 dir
func (p *module) getfile(w http.ResponseWriter, req *http.Request) error {
	http.Redirect(w, req, p.s3.Location(getObjectPath(req)), http.StatusFound)
	return nil
}

func (p *module) deletefile(w http.ResponseWriter, req *http.Request) error {
	return p.s3.Remove(req.Context(), getObjectPath(req))
}

func getObjectPath(req *http.Request) string {
	return strings.TrimPrefix(req.URL.Path, "/s3/")
}
