package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/yubo/apiserver/pkg/config/configtls"
)

type Config struct {
	Endpoint        string                     `json:"endpoint" description:"s3 endpoint, e.g. http://localhost:9000"`
	AccessKeyID     string                     `json:"accessKeyID"`
	SecretAccessKey string                     `json:"secretAccessKey"`
	BucketName      string                     `json:"bucketName"`
	TLS             configtls.TLSClientSetting `json:"tls"`
}

/*
tls:
  insecure: true
  caFile: cafile
  certFile: certfile
  keyFile: keyfile
*/

type S3Client interface {
	Put(ctx context.Context, objectPath, contentType string, reader io.Reader, objectSize int64) error
	Remove(ctx context.Context, objectPath string) error
	Location(objectPath string) string
}

type minioClient struct {
	*minio.Client
	bucketName string
	endpoint   string
}

func New(cf *Config) (S3Client, error) {
	u, err := url.Parse(cf.Endpoint)
	if err != nil {
		return nil, err
	}

	opts := &minio.Options{
		Creds: credentials.NewStaticV4(cf.AccessKeyID, cf.SecretAccessKey, ""),
	}

	if u.Scheme == "https" {
		tlsCfg, err := cf.TLS.LoadTLSConfig()
		if err != nil {
			return nil, err
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = tlsCfg
		opts.Transport = transport
	}

	cli, err := minio.New(u.Host, opts)
	if err != nil {
		return nil, err
	}

	// check
	if ok, err := cli.BucketExists(context.TODO(), cf.BucketName); err != nil || !ok {
		return nil, fmt.Errorf("s3 bucket[%s] does't exist", cf.BucketName)
	}

	return &minioClient{
		Client:     cli,
		endpoint:   fmt.Sprintf("%s://%s/", u.Scheme, u.Host),
		bucketName: cf.BucketName,
	}, nil
}

func (p *minioClient) Put(ctx context.Context, objectPath, contentType string, reader io.Reader, objectSize int64) error {
	_, err := p.PutObject(ctx, p.bucketName, objectPath, reader, objectSize, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (p *minioClient) Remove(ctx context.Context, objectPath string) error {
	return p.RemoveObject(ctx, p.bucketName, objectPath, minio.RemoveObjectOptions{})
}

func (p *minioClient) Location(objectPath string) string {
	return p.endpoint + path.Join(p.bucketName, objectPath)
}
