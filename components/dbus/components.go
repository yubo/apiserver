package dbus

import (
	"context"
	"errors"
	"io"

	"github.com/yubo/apiserver/pkg/audit"
	"github.com/yubo/apiserver/pkg/authentication/authenticatorfactory"
	authUser "github.com/yubo/apiserver/pkg/authentication/user"
	"github.com/yubo/apiserver/pkg/db"
	"github.com/yubo/apiserver/pkg/dynamiccertificates"
	"github.com/yubo/apiserver/pkg/server"
	"google.golang.org/grpc"
)

type key int

const (
	_ key = iota
	passwordfileKey
	dbKey
	s3ClientKey
	apiServerKey
	grpcserverKey
	authnKey              // Authentication
	authzKey              // authorization
	auditKey              // audit
	clientCAKey           // clientCA
	authnRequestHeaderkey // Authentication.RequestHeader
)

// password file
type PasswordfileT interface {
	Authenticate(ctx context.Context, usr, pwd string) authUser.Info
}

func RegisterPasswordfile(o PasswordfileT) {
	MustRegister(passwordfileKey, o)
}
func GetPasswordfile() (PasswordfileT, error) {
	ret, ok := get(passwordfileKey).(PasswordfileT)
	if !ok {
		return nil, errors.New("db not registered")
	}
	return ret, nil
}
func Passwordfile() PasswordfileT {
	ret, err := GetPasswordfile()
	if err != nil {
		panic(err)
	}
	return ret
}

// db
func RegisterDB(i db.DB) {
	MustRegister(dbKey, i)
}
func GetDB() (db.DB, error) {
	ret, ok := get(dbKey).(db.DB)
	if !ok {
		return nil, errors.New("db not registered")
	}
	return ret, nil
}
func DB() db.DB {
	ret, err := GetDB()
	if err != nil {
		panic(err)
	}
	return ret
}

// s3
type S3ClientT interface {
	Put(ctx context.Context, objectPath, contentType string, reader io.Reader, objectSize int64) error
	Remove(ctx context.Context, objectPath string) error
	Location(objectPath string) string
}

func RegisterS3Client(v S3ClientT) {
	MustRegister(s3ClientKey, v)
}
func GetS3Client() (S3ClientT, error) {
	ret, ok := get(s3ClientKey).(S3ClientT)
	if !ok {
		return nil, errors.New("s3 client not registered")
	}

	return ret, nil
}
func S3Client() S3ClientT {
	ret, err := GetS3Client()
	if err != nil {
		panic(err)
	}
	return ret
}

// api/http server
func RegisterAPIServer(i server.APIServer) {
	MustRegister(apiServerKey, i)
}
func GetAPIServer() (server.APIServer, error) {
	ret, ok := get(apiServerKey).(server.APIServer)
	if !ok {
		return nil, errors.New("api server client not registered")
	}
	return ret, nil
}
func APIServer() server.APIServer {
	ret, err := GetAPIServer()
	if err != nil {
		panic(err)
	}
	return ret
}

// grpc server
func RegisterGrpcServer(o *grpc.Server) {
	MustRegister(grpcserverKey, o)
}
func GetGrpcServer() (*grpc.Server, error) {
	ret, ok := get(grpcserverKey).(*grpc.Server)
	if !ok {
		return nil, errors.New("grpc server client not registered")
	}

	return ret, nil
}
func GrpcServer() *grpc.Server {
	ret, err := GetGrpcServer()
	if err != nil {
		panic(err)
	}
	return ret
}

// authn / Authentication
func RegisterAuthenticationInfo(o *server.AuthenticationInfo) {
	MustRegister(authnKey, o)
}
func GetAuthenticationInfo() (*server.AuthenticationInfo, error) {
	ret, ok := get(authnKey).(*server.AuthenticationInfo)
	if !ok {
		return nil, errors.New("AuthenticationInfo not registered")
	}
	return ret, nil
}
func AuthenticationInfo() *server.AuthenticationInfo {
	ret, err := GetAuthenticationInfo()
	if err != nil {
		panic(err)
	}
	return ret
}

// authz // authorizationInfo
func RegisterAuthorizationInfo(o *server.AuthorizationInfo) {
	MustRegister(authzKey, o)
}
func GetAuthorizationInfo() (*server.AuthorizationInfo, error) {
	ret, ok := get(authzKey).(*server.AuthorizationInfo)
	if !ok {
		return nil, errors.New("AuthorizationInfo not registered")
	}
	return ret, nil
}
func AuthorizationInfo() *server.AuthorizationInfo {
	ret, err := GetAuthorizationInfo()
	if err != nil {
		panic(err)
	}
	return ret
}

// audit
type Audit interface {
	Backend() audit.Backend
	AuditPolicyRuleEvaluator() audit.PolicyRuleEvaluator
}

func RegisterAudit(a Audit) {
	MustRegister(auditKey, a)
}
func GetAuditor() (Audit, error) {
	ret, ok := get(auditKey).(Audit)
	if !ok {
		return nil, errors.New("Audit not registered")
	}
	return ret, nil
}

// clientCA
func RegisterClientCA(o dynamiccertificates.CAContentProvider) {
	MustRegister(authzKey, o)
}
func GetClientCA() (dynamiccertificates.CAContentProvider, error) {
	ret, ok := get(authzKey).(dynamiccertificates.CAContentProvider)
	if !ok {
		return nil, errors.New("ClientCA not registered")
	}
	return ret, nil
}
func ClientCA() dynamiccertificates.CAContentProvider {
	ret, err := GetClientCA()
	if err != nil {
		panic(err)
	}
	return ret
}

func RegisterRequestHeaderConfig(o *authenticatorfactory.RequestHeaderConfig) {
	MustRegister(authnRequestHeaderkey, o)
}

func GetRequestHeaderConfig() (*authenticatorfactory.RequestHeaderConfig, error) {
	ret, ok := get(authnRequestHeaderkey).(*authenticatorfactory.RequestHeaderConfig)
	if !ok {
		return nil, errors.New("RequestHeaderConfig not registered")
	}
	return ret, nil
}
