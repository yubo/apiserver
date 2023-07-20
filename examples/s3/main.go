package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/yubo/apiserver/components/cli"
	"github.com/yubo/apiserver/components/dbus"
	"github.com/yubo/apiserver/pkg/proc"
	"github.com/yubo/apiserver/pkg/responsewriters"
	"github.com/yubo/apiserver/pkg/s3"
	"github.com/yubo/golib/scheme"

	_ "github.com/yubo/apiserver/pkg/s3/register"
	_ "github.com/yubo/apiserver/pkg/server/register"
)

type module struct {
	s3 s3.S3Client
}

func main() {
	command := proc.NewRootCmd(proc.WithRun(new(module).start))
	code := cli.Run(command)
	os.Exit(code)
}

func (p *module) start(ctx context.Context) error {
	srv, err := dbus.GetAPIServer()
	if err != nil {
		return err
	}

	s3, err := dbus.GetS3Client()
	if err != nil {
		return err
	}
	p.s3 = s3

	srv.HandlePrefix("/s3/", p)
	return nil
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
		responsewriters.ErrorNegotiated(err, scheme.NegotiatedSerializer, w, req)
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
