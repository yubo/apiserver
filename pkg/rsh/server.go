// +build linux darwin

package rsh

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"

	restful "github.com/emicklei/go-restful"
	"github.com/gorilla/websocket"
	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

type Server struct {
	*RshConfig
	ctx context.Context
}

func NewServer(conf *RshConfig, ctx context.Context) (*Server, error) {
	s := &Server{
		RshConfig: conf,
		ctx:       ctx,
	}

	return s, nil
}

func (p *Server) Handle(req *restful.Request, resp *restful.Response, cmd, env []string) error {
	tty, err := NewRsh(p.RshConfig, nil)
	if err != nil {
		return err
	}

	conn, err := NewWebSocket(req, resp, p.Timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	klog.Infof("Connection created from by %s", conn.RemoteAddr())
	klog.Infof("cmd %s", strings.Join(cmd, " "))
	klog.Infof("env %s", strings.Join(env, " "))
	defer klog.Infof("%s desconnection", conn.RemoteAddr())

	if err := tty.Run(conn, cmd, env); err != nil {
		conn.Error(websocket.CloseInternalServerErr, err.Error())
	}

	return nil
}

func GetRshData(conn *RshConn, req *http.Request) ([]byte, error) {

	// read data pkg first
	length := 0
	if l := req.Header.Get("Rsh-Data-Length"); l != "" {
		length = util.Atoi(l)
	} else if l := req.FormValue("rshDataLength"); l != "" {
		length = util.Atoi(l)
	}
	// klog.V(5).Infof("get rsh hello msg length is %d", length)

	//if length > RshBuffSize {
	//	return nil, status.Errorf(codes.InvalidArgument, "WebSocket Packet Size(%d) exceeds the limit of ReadBufferSize(%d)", length, RshBuffSize)
	//}

	buf := make([]byte, length)
	for i := 0; i < length; {
		if n, err := conn.Read(buf[i:]); err != nil {
			return nil, err
		} else {
			i += n
		}
	}

	return buf, nil
}

func (p *Server) DataHandle(req *restful.Request, resp *restful.Response, cb func(data []byte) (cmd, env []string, err error)) error {
	tty, err := NewRsh(p.RshConfig, nil)
	if err != nil {
		return err
	}

	conn, err := NewWebSocket(req, resp, p.Timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	klog.Infof("Connection created from by %s", conn.RemoteAddr())
	defer klog.Infof("%s desconnection", conn.RemoteAddr())

	// read data pkg first
	length := 0
	if l := req.Request.Header.Get("Rsh-Data-Length"); l != "" {
		length = util.Atoi(l)
	} else if l := req.Request.FormValue("rshDataLength"); l != "" {
		length = util.Atoi(l)
	}

	buf := make([]byte, length)
	for i := 0; i < length; {
		if n, err := conn.Read(buf[i:]); err != nil {
			conn.Error(websocket.CloseInternalServerErr, err.Error())
			return nil
		} else {
			i += n
		}
	}

	cmd, env, err := cb(buf)
	if err != nil {
		conn.Error(websocket.CloseInternalServerErr, err.Error())
		return nil
	}

	klog.V(6).Infof("rsh executing command: %s %s",
		strings.Join(env, " "),
		strings.Join(cmd, " "))

	if err = tty.Run(conn, cmd, env); err != nil {
		conn.Error(websocket.CloseInternalServerErr, err.Error())
	}

	klog.V(6).Infof("leaving rsh.DataHandle, err %v", err)
	return nil
}

func (p *Server) Exec(conn *RshConn, cmd, env []string) error {
	tty, err := NewRsh(p.RshConfig, nil)
	if err != nil {
		return err
	}
	klog.Infof("Connection created from by %s", conn.RemoteAddr())
	defer klog.Infof("%s desconnection", conn.RemoteAddr())

	if klog.V(6).Enabled() {
		klog.InfoDepth(1, fmt.Sprintf("rsh executing command: %s %s",
			strings.Join(env, " "),
			strings.Join(cmd, " ")))
	}

	err = tty.Run(conn, cmd, env)
	klog.V(6).Infof("leaving rsh.Exec, err %v", err)
	return err
}

func (p *Server) Ssh(conn *RshConn, privateKey, hostName string, port int, env []string) error {
	file, err := ioutil.TempFile(p.TmpPath, "ssh_")
	if err != nil {
		return err
	}
	fileName := file.Name()
	defer os.Remove(fileName)

	if _, err := file.Write([]byte(privateKey)); err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	cmd := []string{"/bin/ssh", "-i", fileName}

	if port > 0 {
		cmd = append(cmd, "-p", strconv.Itoa(port))
	}

	cmd = append(cmd, hostName)

	return p.Exec(conn, cmd, env)
}
