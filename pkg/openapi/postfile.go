package openapi

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/yubo/golib/util"
	"k8s.io/klog/v2"
)

var (
	PostFilePrefixPath = "/tmp"
	PostFileReadFile   = util.ReadFile
)

type postFile struct {
	FileName *string `json:"fileName" description:"file name"`
	Data     *string `json:"data" description:"file content, Base64Encode"`
}

type PostFile struct {
	FileName     *string `json:"fileName" description:"file name"`
	Data         *string `json:"data" description:"file content, Base64Encode"`
	OrigFileName *string `json:"-"`
	RawData      []byte  `json:"-"`
}

type ObjectChecker interface {
	IsEmpty() bool
}

func (p PostFile) String() string {
	if p.FileName != nil {
		return *p.FileName
	}
	return ""
}

// key must include "--"
func (p PostFile) CmdArg(key string) []string {
	if key == "" {
		return []string{util.StringValue(p.FileName)}
	}
	return []string{key, util.StringValue(p.FileName)}
}

func (p *PostFile) Set(fileName string) error {
	p.FileName = &fileName
	return nil
}

func (p *PostFile) Type() string {
	return "PostFile"
}

func (p PostFile) MarshalJSON() (b []byte, err error) {
	if p.FileName == nil {
		// for after UnmarshalJSON
		if p.OrigFileName != nil {
			return []byte("\"" + *p.OrigFileName + "\""), nil
		}
		return []byte("null"), nil
	}

	if len(p.RawData) > 0 {
		b = p.RawData
	} else {
		b, err = PostFileReadFile(util.StringValue(p.FileName))
	}

	if err != nil {
		return
	}

	if klog.V(6).Enabled() {
		klog.Infof("PostFile Marshal %s size %d", util.StringValue(p.FileName), len(b))
	}

	return json.Marshal(&postFile{
		FileName: p.FileName,
		Data:     util.String(util.Base64Encode(b)),
	})
}

func (p *PostFile) UnmarshalJSON(data []byte) error {
	// klog.Infof("entering UnmarshalJSON")

	// init
	p.FileName = nil
	p.OrigFileName = nil
	p.Data = nil

	in := &postFile{}
	err := json.Unmarshal(data, in)
	if err != nil {
		return err
	}

	p.OrigFileName = in.FileName

	content, err := util.Base64Decode(util.StringValue(in.Data))
	if err != nil {
		return err
	}

	fileName, err := util.WriteTempFile(PostFilePrefixPath,
		"*."+filepath.Base(util.StringValue(in.FileName)), content)
	if err != nil {
		return err
	}

	if klog.V(6).Enabled() {
		md5, _ := util.FileMd5sum(fileName)
		klog.Infof("PostFile Marshal md5sum %s %s", fileName, md5)
	}

	p.FileName = &fileName
	return nil
}

func (p *PostFile) Clean() error {
	if p == nil || p.FileName == nil {
		return nil
	}
	err := os.Remove(util.StringValue(p.FileName))
	p.FileName = nil
	return err
}

type PostFiles []PostFile

func (p *PostFiles) String() string {
	var buf bytes.Buffer

	if len(*p) == 0 {
		return ""
	}

	for _, v := range *p {
		if v.FileName != nil {
			buf.WriteString("," + util.StringValue(v.FileName))
		}
	}

	return string(buf.Bytes()[1:])
}

func (p *PostFiles) Set(fileName string) error {
	if p == nil {
		p = &PostFiles{}
	}
	*p = append(*p, PostFile{FileName: util.String(fileName)})
	return nil
}

func (p *PostFiles) Type() string {
	return "PostFiles"
}

func (p *PostFiles) Clean() error {
	if p == nil {
		return nil
	}
	for k, _ := range *p {
		(*p)[k].Clean()
	}
	return nil
}

func (p *PostFiles) CmdArg(key string) []string {
	args := []string{}

	if key == "" {
		for _, v := range *p {
			if v.FileName != nil {
				args = append(args, util.StringValue(v.FileName))
			}
		}
		return args
	}

	for _, v := range *p {
		args = append(args, key, util.StringValue(v.FileName))
	}
	return args
}
