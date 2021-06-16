package v1

import (
	metav1 "github.com/yubo/apiserver/pkg/api/meta/v1"
	"github.com/yubo/golib/staging/types"
)

// Secret holds secret data of a certain type. The total bytes of the values in
// the Data field must be less than MaxSecretSize bytes.
type Secret struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Immutable, if set to true, ensures that data stored in the Secret cannot
	// be updated (only object metadata can be modified).
	// If not set to true, the field can be modified at any time.
	// Defaulted to nil.
	// +optional
	Immutable *bool `json:"immutable,omitempty" protobuf:"varint,5,opt,name=immutable"`

	// Data contains the secret data. Each key must consist of alphanumeric
	// characters, '-', '_' or '.'. The serialized form of the secret data is a
	// base64 encoded string, representing the arbitrary (possibly non-string)
	// data value here. Described in https://tools.ietf.org/html/rfc4648#section-4
	// +optional
	Data map[string][]byte `json:"data,omitempty" protobuf:"bytes,2,rep,name=data"`

	// stringData allows specifying non-binary secret data in string form.
	// It is provided as a write-only input field for convenience.
	// All keys and values are merged into the data field on write, overwriting any existing values.
	// The stringData field is never output when reading from the API.
	// +k8s:conversion-gen=false
	// +optional
	StringData map[string]string `json:"stringData,omitempty" protobuf:"bytes,4,rep,name=stringData"`

	// Used to facilitate programmatic handling of secret data.
	// +optional
	Type SecretType `json:"type,omitempty" protobuf:"bytes,3,opt,name=type,casttype=SecretType"`
}

const MaxSecretSize = 1 * 1024 * 1024

type SecretType string

const (
	// SecretTypeOpaque is the default. Arbitrary user-defined data
	SecretTypeOpaque SecretType = "Opaque"

	// SecretTypeServiceAccountToken contains a token that identifies a service account to the API
	//
	// Required fields:
	// - Secret.Annotations["kubernetes.io/service-account.name"] - the name of the ServiceAccount the token identifies
	// - Secret.Annotations["kubernetes.io/service-account.uid"] - the UID of the ServiceAccount the token identifies
	// - Secret.Data["token"] - a token that identifies the service account to the API
	SecretTypeServiceAccountToken SecretType = "kubernetes.io/service-account-token"

	// ServiceAccountNameKey is the key of the required annotation for SecretTypeServiceAccountToken secrets
	ServiceAccountNameKey = "kubernetes.io/service-account.name"
	// ServiceAccountUIDKey is the key of the required annotation for SecretTypeServiceAccountToken secrets
	ServiceAccountUIDKey = "kubernetes.io/service-account.uid"
	// ServiceAccountTokenKey is the key of the required data for SecretTypeServiceAccountToken secrets
	ServiceAccountTokenKey = "token"
	// ServiceAccountKubeconfigKey is the key of the optional kubeconfig data for SecretTypeServiceAccountToken secrets
	ServiceAccountKubeconfigKey = "kubernetes.kubeconfig"
	// ServiceAccountRootCAKey is the key of the optional root certificate authority for SecretTypeServiceAccountToken secrets
	ServiceAccountRootCAKey = "ca.crt"
	// ServiceAccountNamespaceKey is the key of the optional namespace to use as the default for namespaced API calls
	ServiceAccountNamespaceKey = "namespace"

	// SecretTypeDockercfg contains a dockercfg file that follows the same format rules as ~/.dockercfg
	//
	// Required fields:
	// - Secret.Data[".dockercfg"] - a serialized ~/.dockercfg file
	SecretTypeDockercfg SecretType = "kubernetes.io/dockercfg"

	// DockerConfigKey is the key of the required data for SecretTypeDockercfg secrets
	DockerConfigKey = ".dockercfg"

	// SecretTypeDockerConfigJson contains a dockercfg file that follows the same format rules as ~/.docker/config.json
	//
	// Required fields:
	// - Secret.Data[".dockerconfigjson"] - a serialized ~/.docker/config.json file
	SecretTypeDockerConfigJson SecretType = "kubernetes.io/dockerconfigjson"

	// DockerConfigJsonKey is the key of the required data for SecretTypeDockerConfigJson secrets
	DockerConfigJsonKey = ".dockerconfigjson"

	// SecretTypeBasicAuth contains data needed for basic authentication.
	//
	// Required at least one of fields:
	// - Secret.Data["username"] - username used for authentication
	// - Secret.Data["password"] - password or token needed for authentication
	SecretTypeBasicAuth SecretType = "kubernetes.io/basic-auth"

	// BasicAuthUsernameKey is the key of the username for SecretTypeBasicAuth secrets
	BasicAuthUsernameKey = "username"
	// BasicAuthPasswordKey is the key of the password or token for SecretTypeBasicAuth secrets
	BasicAuthPasswordKey = "password"

	// SecretTypeSSHAuth contains data needed for SSH authetication.
	//
	// Required field:
	// - Secret.Data["ssh-privatekey"] - private SSH key needed for authentication
	SecretTypeSSHAuth SecretType = "kubernetes.io/ssh-auth"

	// SSHAuthPrivateKey is the key of the required SSH private key for SecretTypeSSHAuth secrets
	SSHAuthPrivateKey = "ssh-privatekey"
	// SecretTypeTLS contains information about a TLS client or server secret. It
	// is primarily used with TLS termination of the Ingress resource, but may be
	// used in other types.
	//
	// Required fields:
	// - Secret.Data["tls.key"] - TLS private key.
	//   Secret.Data["tls.crt"] - TLS certificate.
	// TODO: Consider supporting different formats, specifying CA/destinationCA.
	SecretTypeTLS SecretType = "kubernetes.io/tls"

	// TLSCertKey is the key for tls certificates in a TLS secert.
	TLSCertKey = "tls.crt"
	// TLSPrivateKeyKey is the key for the private key field in a TLS secret.
	TLSPrivateKeyKey = "tls.key"
	// SecretTypeBootstrapToken is used during the automated bootstrap process (first
	// implemented by kubeadm). It stores tokens that are used to sign well known
	// ConfigMaps. They are used for authn.
	SecretTypeBootstrapToken SecretType = "bootstrap.kubernetes.io/token"
)

// +k8s:deepcopy-gen:interfaces=github.com/yubo/apiserver/staging/runtime.Object

// SecretList is a list of Secret.
type SecretList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Items is a list of secret objects.
	// More info: https://kubernetes.io/docs/concepts/configuration/secret
	Items []Secret `json:"items" protobuf:"bytes,2,rep,name=items"`
}

// +genclient
// +genclient:method=CreateToken,verb=create,subresource=token,input=github.com/yubo/apiserver/pkg/api/authentication/v1.TokenRequest,result=github.com/yubo/apiserver/pkg/api/authentication/v1.TokenRequest
// +k8s:deepcopy-gen:interfaces=github.com/yubo/apiserver/staging/runtime.Object

// ServiceAccount binds together:
// * a name, understood by users, and perhaps by peripheral systems, for an identity
// * a principal that can be authenticated and authorized
// * a set of secrets
type ServiceAccount struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Secrets is the list of secrets allowed to be used by pods running using this ServiceAccount.
	// More info: https://kubernetes.io/docs/concepts/configuration/secret
	// +optional
	// +patchMergeKey=name
	// +patchStrategy=merge
	Secrets []ObjectReference `json:"secrets,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=secrets"`

	// ImagePullSecrets is a list of references to secrets in the same namespace to use for pulling any images
	// in pods that reference this ServiceAccount. ImagePullSecrets are distinct from Secrets because Secrets
	// can be mounted in the pod, but ImagePullSecrets are only accessed by the kubelet.
	// More info: https://kubernetes.io/docs/concepts/containers/images/#specifying-imagepullsecrets-on-a-pod
	// +optional
	ImagePullSecrets []LocalObjectReference `json:"imagePullSecrets,omitempty" protobuf:"bytes,3,rep,name=imagePullSecrets"`

	// AutomountServiceAccountToken indicates whether pods running as this service account should have an API token automatically mounted.
	// Can be overridden at the pod level.
	// +optional
	AutomountServiceAccountToken *bool `json:"automountServiceAccountToken,omitempty" protobuf:"varint,4,opt,name=automountServiceAccountToken"`
}

// +k8s:deepcopy-gen:interfaces=github.com/yubo/apiserver/staging/runtime.Object

// ServiceAccountList is a list of ServiceAccount objects
type ServiceAccountList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	// +optional
	metav1.ListMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// List of ServiceAccounts.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	Items []ServiceAccount `json:"items" protobuf:"bytes,2,rep,name=items"`
}

type ObjectReference struct {
	// Kind of the referent.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	// +optional
	Kind string `json:"kind,omitempty" protobuf:"bytes,1,opt,name=kind"`
	// Namespace of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
	// +optional
	Namespace string `json:"namespace,omitempty" protobuf:"bytes,2,opt,name=namespace"`
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
	// UID of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#uids
	// +optional
	UID types.UID `json:"uid,omitempty" protobuf:"bytes,4,opt,name=uid,casttype=github.com/yubo/golib/staging/types.UID"`
	// API version of the referent.
	// +optional
	APIVersion string `json:"apiVersion,omitempty" protobuf:"bytes,5,opt,name=apiVersion"`
	// Specific resourceVersion to which this reference is made, if any.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#concurrency-control-and-consistency
	// +optional
	ResourceVersion string `json:"resourceVersion,omitempty" protobuf:"bytes,6,opt,name=resourceVersion"`

	// If referring to a piece of an object instead of an entire object, this string
	// should contain a valid JSON/Go field access statement, such as desiredState.manifest.containers[2].
	// For example, if the object reference is to a container within a pod, this would take on a value like:
	// "spec.containers{name}" (where "name" refers to the name of the container that triggered
	// the event) or if no container name is specified "spec.containers[2]" (container with
	// index 2 in this pod). This syntax is chosen only to have some well-defined way of
	// referencing a part of an object.
	// TODO: this design is not final and this field is subject to change in the future.
	// +optional
	FieldPath string `json:"fieldPath,omitempty" protobuf:"bytes,7,opt,name=fieldPath"`
}

// LocalObjectReference contains enough information to let you locate the
// referenced object inside the same namespace.
type LocalObjectReference struct {
	// Name of the referent.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	// TODO: Add other useful fields. apiVersion, kind, uid?
	// +optional
	Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
}
