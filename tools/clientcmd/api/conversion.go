/*
Copyright 2014 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package api

import (
	"fmt"
	"sort"
	unsafe "unsafe"

	v1 "github.com/yubo/apiserver/tools/clientcmd/api/v1"
	"github.com/yubo/golib/runtime"
)

func Convert_Slice_v1_NamedCluster_To_Map_string_To_Pointer_api_Cluster(in *[]v1.NamedCluster, out *map[string]*Cluster) error {
	for _, curr := range *in {
		newCluster := NewCluster()
		if err := Convert_v1_Cluster_To_api_Cluster(&curr.Cluster, newCluster); err != nil {
			return err
		}
		if *out == nil {
			*out = make(map[string]*Cluster)
		}
		if (*out)[curr.Name] == nil {
			(*out)[curr.Name] = newCluster
		} else {
			return fmt.Errorf("error converting *[]NamedCluster into *map[string]*Cluster: duplicate name \"%v\" in list: %v", curr.Name, *in)
		}
	}
	return nil
}

func Convert_Map_string_To_Pointer_api_Cluster_To_Slice_v1_NamedCluster(in *map[string]*Cluster, out *[]v1.NamedCluster) error {
	allKeys := make([]string, 0, len(*in))
	for key := range *in {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)

	for _, key := range allKeys {
		newCluster := (*in)[key]
		oldCluster := v1.Cluster{}
		if err := Convert_api_Cluster_To_v1_Cluster(newCluster, &oldCluster); err != nil {
			return err
		}
		namedCluster := v1.NamedCluster{Name: key, Cluster: oldCluster}
		*out = append(*out, namedCluster)
	}
	return nil
}

func Convert_Slice_v1_NamedAuthInfo_To_Map_string_To_Pointer_api_AuthInfo(in *[]v1.NamedAuthInfo, out *map[string]*AuthInfo) error {
	for _, curr := range *in {
		newAuthInfo := NewAuthInfo()
		if err := Convert_v1_AuthInfo_To_api_AuthInfo(&curr.AuthInfo, newAuthInfo); err != nil {
			return err
		}
		if *out == nil {
			*out = make(map[string]*AuthInfo)
		}
		if (*out)[curr.Name] == nil {
			(*out)[curr.Name] = newAuthInfo
		} else {
			return fmt.Errorf("error converting *[]NamedAuthInfo into *map[string]*AuthInfo: duplicate name \"%v\" in list: %v", curr.Name, *in)
		}
	}
	return nil
}

func Convert_Map_string_To_Pointer_api_AuthInfo_To_Slice_v1_NamedAuthInfo(in *map[string]*AuthInfo, out *[]v1.NamedAuthInfo) error {
	allKeys := make([]string, 0, len(*in))
	for key := range *in {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)

	for _, key := range allKeys {
		newAuthInfo := (*in)[key]
		oldAuthInfo := v1.AuthInfo{}
		if err := Convert_api_AuthInfo_To_v1_AuthInfo(newAuthInfo, &oldAuthInfo); err != nil {
			return err
		}
		namedAuthInfo := v1.NamedAuthInfo{Name: key, AuthInfo: oldAuthInfo}
		*out = append(*out, namedAuthInfo)
	}
	return nil
}

func Convert_Slice_v1_NamedContext_To_Map_string_To_Pointer_api_Context(in *[]v1.NamedContext, out *map[string]*Context) error {
	for _, curr := range *in {
		newContext := NewContext()
		if err := Convert_v1_Context_To_api_Context(&curr.Context, newContext); err != nil {
			return err
		}
		if *out == nil {
			*out = make(map[string]*Context)
		}
		if (*out)[curr.Name] == nil {
			(*out)[curr.Name] = newContext
		} else {
			return fmt.Errorf("error converting *[]NamedContext into *map[string]*Context: duplicate name \"%v\" in list: %v", curr.Name, *in)
		}
	}
	return nil
}

func Convert_Map_string_To_Pointer_api_Context_To_Slice_v1_NamedContext(in *map[string]*Context, out *[]v1.NamedContext) error {
	allKeys := make([]string, 0, len(*in))
	for key := range *in {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)

	for _, key := range allKeys {
		newContext := (*in)[key]
		oldContext := v1.Context{}
		if err := Convert_api_Context_To_v1_Context(newContext, &oldContext); err != nil {
			return err
		}
		namedContext := v1.NamedContext{Name: key, Context: oldContext}
		*out = append(*out, namedContext)
	}
	return nil
}

func Convert_Slice_v1_NamedExtension_To_Map_string_To_runtime_Object(in *[]v1.NamedExtension, out *map[string]runtime.Object) error {
	for _, curr := range *in {
		var newExtension runtime.Object
		if err := runtime.Convert_runtime_RawExtension_To_runtime_Object(&curr.Extension, &newExtension); err != nil {
			return err
		}
		if *out == nil {
			*out = make(map[string]runtime.Object)
		}
		if (*out)[curr.Name] == nil {
			(*out)[curr.Name] = newExtension
		} else {
			return fmt.Errorf("error converting *[]NamedExtension into *map[string]runtime.Object: duplicate name \"%v\" in list: %v", curr.Name, *in)
		}
	}
	return nil
}

func Convert_Map_string_To_runtime_Object_To_Slice_v1_NamedExtension(in *map[string]runtime.Object, out *[]v1.NamedExtension) error {
	allKeys := make([]string, 0, len(*in))
	for key := range *in {
		allKeys = append(allKeys, key)
	}
	sort.Strings(allKeys)

	for _, key := range allKeys {
		newExtension := (*in)[key]
		oldExtension := runtime.RawExtension{}
		if err := runtime.Convert_runtime_Object_To_runtime_RawExtension(&newExtension, &oldExtension); err != nil {
			return nil
		}
		namedExtension := v1.NamedExtension{Name: key, Extension: oldExtension}
		*out = append(*out, namedExtension)
	}
	return nil
}

func autoConvert_v1_AuthInfo_To_api_AuthInfo(in *v1.AuthInfo, out *AuthInfo) error {
	out.ClientCertificate = in.ClientCertificate
	out.ClientCertificateData = *(*[]byte)(unsafe.Pointer(&in.ClientCertificateData))
	out.ClientKey = in.ClientKey
	out.ClientKeyData = *(*[]byte)(unsafe.Pointer(&in.ClientKeyData))
	out.Token = in.Token
	out.TokenFile = in.TokenFile
	out.Impersonate = in.Impersonate
	out.ImpersonateGroups = *(*[]string)(unsafe.Pointer(&in.ImpersonateGroups))
	out.ImpersonateUserExtra = *(*map[string][]string)(unsafe.Pointer(&in.ImpersonateUserExtra))
	out.Username = in.Username
	out.Password = in.Password
	out.AuthProvider = (*AuthProviderConfig)(unsafe.Pointer(in.AuthProvider))
	if in.Exec != nil {
		in, out := &in.Exec, &out.Exec
		*out = new(ExecConfig)
		if err := Convert_v1_ExecConfig_To_api_ExecConfig(*in, *out); err != nil {
			return err
		}
	} else {
		out.Exec = nil
	}
	if err := Convert_Slice_v1_NamedExtension_To_Map_string_To_runtime_Object(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_v1_AuthInfo_To_api_AuthInfo is an autogenerated conversion function.
func Convert_v1_AuthInfo_To_api_AuthInfo(in *v1.AuthInfo, out *AuthInfo) error {
	return autoConvert_v1_AuthInfo_To_api_AuthInfo(in, out)
}

func autoConvert_api_AuthInfo_To_v1_AuthInfo(in *AuthInfo, out *v1.AuthInfo) error {
	// INFO: in.LocationOfOrigin opted out of conversion generation
	out.ClientCertificate = in.ClientCertificate
	out.ClientCertificateData = *(*[]byte)(unsafe.Pointer(&in.ClientCertificateData))
	out.ClientKey = in.ClientKey
	out.ClientKeyData = *(*[]byte)(unsafe.Pointer(&in.ClientKeyData))
	out.Token = in.Token
	out.TokenFile = in.TokenFile
	out.Impersonate = in.Impersonate
	out.ImpersonateGroups = *(*[]string)(unsafe.Pointer(&in.ImpersonateGroups))
	out.ImpersonateUserExtra = *(*map[string][]string)(unsafe.Pointer(&in.ImpersonateUserExtra))
	out.Username = in.Username
	out.Password = in.Password
	out.AuthProvider = (*v1.AuthProviderConfig)(unsafe.Pointer(in.AuthProvider))
	if in.Exec != nil {
		in, out := &in.Exec, &out.Exec
		*out = new(v1.ExecConfig)
		if err := Convert_api_ExecConfig_To_v1_ExecConfig(*in, *out); err != nil {
			return err
		}
	} else {
		out.Exec = nil
	}
	if err := Convert_Map_string_To_runtime_Object_To_Slice_v1_NamedExtension(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_api_AuthInfo_To_v1_AuthInfo is an autogenerated conversion function.
func Convert_api_AuthInfo_To_v1_AuthInfo(in *AuthInfo, out *v1.AuthInfo) error {
	return autoConvert_api_AuthInfo_To_v1_AuthInfo(in, out)
}

func autoConvert_v1_AuthProviderConfig_To_api_AuthProviderConfig(in *v1.AuthProviderConfig, out *AuthProviderConfig) error {
	out.Name = in.Name
	out.Config = *(*map[string]string)(unsafe.Pointer(&in.Config))
	return nil
}

// Convert_v1_AuthProviderConfig_To_api_AuthProviderConfig is an autogenerated conversion function.
func Convert_v1_AuthProviderConfig_To_api_AuthProviderConfig(in *v1.AuthProviderConfig, out *AuthProviderConfig) error {
	return autoConvert_v1_AuthProviderConfig_To_api_AuthProviderConfig(in, out)
}

func autoConvert_api_AuthProviderConfig_To_v1_AuthProviderConfig(in *AuthProviderConfig, out *v1.AuthProviderConfig) error {
	out.Name = in.Name
	out.Config = *(*map[string]string)(unsafe.Pointer(&in.Config))
	return nil
}

// Convert_api_AuthProviderConfig_To_v1_AuthProviderConfig is an autogenerated conversion function.
func Convert_api_AuthProviderConfig_To_v1_AuthProviderConfig(in *AuthProviderConfig, out *v1.AuthProviderConfig) error {
	return autoConvert_api_AuthProviderConfig_To_v1_AuthProviderConfig(in, out)
}

func autoConvert_v1_Cluster_To_api_Cluster(in *v1.Cluster, out *Cluster) error {
	out.Server = in.Server
	out.TLSServerName = in.TLSServerName
	out.InsecureSkipTLSVerify = in.InsecureSkipTLSVerify
	out.CertificateAuthority = in.CertificateAuthority
	out.CertificateAuthorityData = *(*[]byte)(unsafe.Pointer(&in.CertificateAuthorityData))
	out.ProxyURL = in.ProxyURL
	if err := Convert_Slice_v1_NamedExtension_To_Map_string_To_runtime_Object(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_v1_Cluster_To_api_Cluster is an autogenerated conversion function.
func Convert_v1_Cluster_To_api_Cluster(in *v1.Cluster, out *Cluster) error {
	return autoConvert_v1_Cluster_To_api_Cluster(in, out)
}

func autoConvert_api_Cluster_To_v1_Cluster(in *Cluster, out *v1.Cluster) error {
	// INFO: in.LocationOfOrigin opted out of conversion generation
	out.Server = in.Server
	out.TLSServerName = in.TLSServerName
	out.InsecureSkipTLSVerify = in.InsecureSkipTLSVerify
	out.CertificateAuthority = in.CertificateAuthority
	out.CertificateAuthorityData = *(*[]byte)(unsafe.Pointer(&in.CertificateAuthorityData))
	out.ProxyURL = in.ProxyURL
	if err := Convert_Map_string_To_runtime_Object_To_Slice_v1_NamedExtension(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_api_Cluster_To_v1_Cluster is an autogenerated conversion function.
func Convert_api_Cluster_To_v1_Cluster(in *Cluster, out *v1.Cluster) error {
	return autoConvert_api_Cluster_To_v1_Cluster(in, out)
}

func autoConvert_v1_Config_To_api_Config(in *v1.Config, out *Config) error {
	// INFO: in.Kind opted out of conversion generation
	// INFO: in.APIVersion opted out of conversion generation
	if err := Convert_v1_Preferences_To_api_Preferences(&in.Preferences, &out.Preferences); err != nil {
		return err
	}
	if err := Convert_Slice_v1_NamedCluster_To_Map_string_To_Pointer_api_Cluster(&in.Clusters, &out.Clusters); err != nil {
		return err
	}
	if err := Convert_Slice_v1_NamedAuthInfo_To_Map_string_To_Pointer_api_AuthInfo(&in.AuthInfos, &out.AuthInfos); err != nil {
		return err
	}
	if err := Convert_Slice_v1_NamedContext_To_Map_string_To_Pointer_api_Context(&in.Contexts, &out.Contexts); err != nil {
		return err
	}
	out.CurrentContext = in.CurrentContext
	if err := Convert_Slice_v1_NamedExtension_To_Map_string_To_runtime_Object(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_v1_Config_To_api_Config is an autogenerated conversion function.
func Convert_v1_Config_To_api_Config(in *v1.Config, out *Config) error {
	return autoConvert_v1_Config_To_api_Config(in, out)
}

func autoConvert_api_Config_To_v1_Config(in *Config, out *v1.Config) error {
	// INFO: in.Kind opted out of conversion generation
	// INFO: in.APIVersion opted out of conversion generation
	if err := Convert_api_Preferences_To_v1_Preferences(&in.Preferences, &out.Preferences); err != nil {
		return err
	}
	if err := Convert_Map_string_To_Pointer_api_Cluster_To_Slice_v1_NamedCluster(&in.Clusters, &out.Clusters); err != nil {
		return err
	}
	if err := Convert_Map_string_To_Pointer_api_AuthInfo_To_Slice_v1_NamedAuthInfo(&in.AuthInfos, &out.AuthInfos); err != nil {
		return err
	}
	if err := Convert_Map_string_To_Pointer_api_Context_To_Slice_v1_NamedContext(&in.Contexts, &out.Contexts); err != nil {
		return err
	}
	out.CurrentContext = in.CurrentContext
	if err := Convert_Map_string_To_runtime_Object_To_Slice_v1_NamedExtension(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_api_Config_To_v1_Config is an autogenerated conversion function.
func Convert_api_Config_To_v1_Config(in *Config, out *v1.Config) error {
	return autoConvert_api_Config_To_v1_Config(in, out)
}

func autoConvert_v1_Context_To_api_Context(in *v1.Context, out *Context) error {
	out.Cluster = in.Cluster
	out.AuthInfo = in.AuthInfo
	out.Namespace = in.Namespace
	if err := Convert_Slice_v1_NamedExtension_To_Map_string_To_runtime_Object(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_v1_Context_To_api_Context is an autogenerated conversion function.
func Convert_v1_Context_To_api_Context(in *v1.Context, out *Context) error {
	return autoConvert_v1_Context_To_api_Context(in, out)
}

func autoConvert_api_Context_To_v1_Context(in *Context, out *v1.Context) error {
	// INFO: in.LocationOfOrigin opted out of conversion generation
	out.Cluster = in.Cluster
	out.AuthInfo = in.AuthInfo
	out.Namespace = in.Namespace
	if err := Convert_Map_string_To_runtime_Object_To_Slice_v1_NamedExtension(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_api_Context_To_v1_Context is an autogenerated conversion function.
func Convert_api_Context_To_v1_Context(in *Context, out *v1.Context) error {
	return autoConvert_api_Context_To_v1_Context(in, out)
}

func autoConvert_v1_ExecConfig_To_api_ExecConfig(in *v1.ExecConfig, out *ExecConfig) error {
	out.Command = in.Command
	out.Args = *(*[]string)(unsafe.Pointer(&in.Args))
	out.Env = *(*[]ExecEnvVar)(unsafe.Pointer(&in.Env))
	out.APIVersion = in.APIVersion
	out.InstallHint = in.InstallHint
	out.ProvideClusterInfo = in.ProvideClusterInfo
	return nil
}

// Convert_v1_ExecConfig_To_api_ExecConfig is an autogenerated conversion function.
func Convert_v1_ExecConfig_To_api_ExecConfig(in *v1.ExecConfig, out *ExecConfig) error {
	return autoConvert_v1_ExecConfig_To_api_ExecConfig(in, out)
}

func autoConvert_api_ExecConfig_To_v1_ExecConfig(in *ExecConfig, out *v1.ExecConfig) error {
	out.Command = in.Command
	out.Args = *(*[]string)(unsafe.Pointer(&in.Args))
	out.Env = *(*[]v1.ExecEnvVar)(unsafe.Pointer(&in.Env))
	out.APIVersion = in.APIVersion
	out.InstallHint = in.InstallHint
	out.ProvideClusterInfo = in.ProvideClusterInfo
	// INFO: in.Config opted out of conversion generation
	return nil
}

// Convert_api_ExecConfig_To_v1_ExecConfig is an autogenerated conversion function.
func Convert_api_ExecConfig_To_v1_ExecConfig(in *ExecConfig, out *v1.ExecConfig) error {
	return autoConvert_api_ExecConfig_To_v1_ExecConfig(in, out)
}

func autoConvert_v1_ExecEnvVar_To_api_ExecEnvVar(in *v1.ExecEnvVar, out *ExecEnvVar) error {
	out.Name = in.Name
	out.Value = in.Value
	return nil
}

// Convert_v1_ExecEnvVar_To_api_ExecEnvVar is an autogenerated conversion function.
func Convert_v1_ExecEnvVar_To_api_ExecEnvVar(in *v1.ExecEnvVar, out *ExecEnvVar) error {
	return autoConvert_v1_ExecEnvVar_To_api_ExecEnvVar(in, out)
}

func autoConvert_api_ExecEnvVar_To_v1_ExecEnvVar(in *ExecEnvVar, out *v1.ExecEnvVar) error {
	out.Name = in.Name
	out.Value = in.Value
	return nil
}

// Convert_api_ExecEnvVar_To_v1_ExecEnvVar is an autogenerated conversion function.
func Convert_api_ExecEnvVar_To_v1_ExecEnvVar(in *ExecEnvVar, out *v1.ExecEnvVar) error {
	return autoConvert_api_ExecEnvVar_To_v1_ExecEnvVar(in, out)
}

func autoConvert_v1_Preferences_To_api_Preferences(in *v1.Preferences, out *Preferences) error {
	out.Colors = in.Colors
	if err := Convert_Slice_v1_NamedExtension_To_Map_string_To_runtime_Object(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_v1_Preferences_To_api_Preferences is an autogenerated conversion function.
func Convert_v1_Preferences_To_api_Preferences(in *v1.Preferences, out *Preferences) error {
	return autoConvert_v1_Preferences_To_api_Preferences(in, out)
}

func autoConvert_api_Preferences_To_v1_Preferences(in *Preferences, out *v1.Preferences) error {
	out.Colors = in.Colors
	if err := Convert_Map_string_To_runtime_Object_To_Slice_v1_NamedExtension(&in.Extensions, &out.Extensions); err != nil {
		return err
	}
	return nil
}

// Convert_api_Preferences_To_v1_Preferences is an autogenerated conversion function.
func Convert_api_Preferences_To_v1_Preferences(in *Preferences, out *v1.Preferences) error {
	return autoConvert_api_Preferences_To_v1_Preferences(in, out)
}
