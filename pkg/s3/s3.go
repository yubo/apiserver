package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/yubo/apiserver/pkg/config/configtls"
	"k8s.io/klog/v2"
)

type Config struct {
	Endpoint        string                     `json:"endpoint" description:"s3 endpoint, e.g. 127.0.0.1:9000"`
	ExternAddress   string                     `json:"externAddress" description:"s3 extern address, e.g. http://127.0.0.1:9000"`
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
	bucketName    string
	endpoint      string
	externAddress string
}

func New(cf *Config) (S3Client, error) {
	opts := &minio.Options{
		Creds: credentials.NewStaticV4(cf.AccessKeyID, cf.SecretAccessKey, ""),
	}

	client := &minioClient{
		bucketName:    cf.BucketName,
		endpoint:      fmt.Sprintf("http://%s/", cf.Endpoint),
		externAddress: strings.TrimRight(cf.ExternAddress, "/") + "/",
	}

	if cf.TLS.Insecure {
		opts.Secure = true

		tlsCfg, err := cf.TLS.LoadTLSConfig()
		if err != nil {
			return nil, err
		}

		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = tlsCfg
		opts.Transport = transport

		client.endpoint = fmt.Sprintf("https://%s/", cf.Endpoint)
	}

	if client.externAddress == "" {
		client.externAddress = client.endpoint
	}

	cli, err := minio.New(cf.Endpoint, opts)
	if err != nil {
		return nil, err
	}

	// check
	if ok, err := cli.BucketExists(context.TODO(), cf.BucketName); err != nil || !ok {
		klog.InfoS("bucketexists", "err", err)
		return nil, fmt.Errorf("s3 bucket[%s] does't exist", cf.BucketName)
	}

	client.Client = cli

	return client, nil
}

func (p *minioClient) Put(ctx context.Context, objectPath, contentType string, reader io.Reader, objectSize int64) error {
	_, err := p.PutObject(ctx, p.bucketName, objectPath, reader, objectSize, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (p *minioClient) Remove(ctx context.Context, objectPath string) error {
	return p.RemoveObject(ctx, p.bucketName, objectPath, minio.RemoveObjectOptions{})
}

func (p *minioClient) Location(objectPath string) string {
	return p.externAddress + path.Join(p.bucketName, objectPath)
}
