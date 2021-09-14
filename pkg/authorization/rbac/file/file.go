package file

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"

	apirbac "github.com/yubo/apiserver/pkg/api/rbac"
	"github.com/yubo/apiserver/pkg/authorization/rbac"
	"github.com/yubo/golib/api"
	"github.com/yubo/golib/runtime"
	"github.com/yubo/golib/util/yaml"
	"k8s.io/klog/v2"
)

type Config struct {
	ConfigPath string `json:"configPath" flag:"rbac-config-path" description:"RBAC config path"`
}

type FileStorage struct {
	config              *Config
	roles               []*apirbac.Role
	roleBindings        []*apirbac.RoleBinding
	clusterRoles        []*apirbac.ClusterRole
	clusterRoleBindings []*apirbac.ClusterRoleBinding
}

func NewRBAC(config *Config) (*rbac.RBACAuthorizer, error) {
	f, err := NewFileStorage(config)
	if err != nil {
		return nil, err
	}
	return rbac.New(
		&rbac.RoleGetter{Lister: NewRoleLister(f)},
		&rbac.RoleBindingLister{Lister: NewRoleBindingLister(f)},
		&rbac.ClusterRoleGetter{Lister: NewClusterRoleLister(f)},
		&rbac.ClusterRoleBindingLister{Lister: NewClusterRoleBindingLister(f)},
	), nil
}

func NewFileStorage(config *Config) (*FileStorage, error) {
	klog.V(10).Infof("rbac.file entering")
	f := &FileStorage{config: config}

	err := f.loadDir(config.ConfigPath, 1)
	if err != nil {
		return nil, err
	}

	f.sort()

	defer klog.V(10).InfoS("rbac.file leaving",
		"Role", len(f.roles),
		"RoleBinding", len(f.roleBindings),
		"ClusterRole", len(f.clusterRoles),
		"ClusterRoleBinding", len(f.clusterRoleBindings),
	)
	return f, nil
}

// maxDepth = maxDepth <= 1 ? 1 : maxDepth
func (p *FileStorage) loadDir(path string, maxDepth int) error {
	entries, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if maxDepth > 1 {
				if err := p.loadDir(filepath.Join(path,
					entry.Name()), maxDepth-1); err != nil {
					return err
				}
			}
			continue
		}

		p.loadFile(entry, path)
	}
	return nil
}

func (p *FileStorage) loadFile(file os.FileInfo, path string) error {
	fileName := file.Name()
	ext := filepath.Ext(fileName)
	absPath := filepath.Join(path, fileName)

	if ext != ".yaml" && ext != ".yml" {
		klog.V(6).Infof("Skipping file: %s", absPath)
		return nil
	}

	fd, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer fd.Close()

	d := yaml.NewYAMLOrJSONDecoder(fd, 4096)
	for {
		ext := runtime.RawExtension{}
		if err := d.Decode(&ext); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("error parsing %s: %v", absPath, err)
		}
		ext.Raw = bytes.TrimSpace(ext.Raw)
		if len(ext.Raw) == 0 || bytes.Equal(ext.Raw, []byte("null")) {
			continue
		}

		if err := p.loadConfig(ext.Raw, absPath); err != nil {
			return err
		}
	}
}

func (p *FileStorage) loadConfig(data []byte, source string) error {
	meta := &api.TypeMeta{}
	if err := yaml.Unmarshal(data, meta); err != nil {
		return fmt.Errorf("error parsing %s: %v", source, err)
	}

	switch meta.Kind {
	case "Role":
		obj := &apirbac.Role{}
		if err := yaml.Unmarshal(data, obj); err != nil {
			return fmt.Errorf("error parsing %s: %v", source, err)
		}
		p.roles = append(p.roles, obj)
	case "RoleBinding":
		obj := &apirbac.RoleBinding{}
		if err := yaml.Unmarshal(data, obj); err != nil {
			return fmt.Errorf("error parsing %s: %v", source, err)
		}
		p.roleBindings = append(p.roleBindings, obj)
	case "ClusterRole":
		obj := &apirbac.ClusterRole{}
		if err := yaml.Unmarshal(data, obj); err != nil {
			return fmt.Errorf("error parsing %s: %v", source, err)
		}
		p.clusterRoles = append(p.clusterRoles, obj)
	case "ClusterRoleBinding":
		obj := &apirbac.ClusterRoleBinding{}
		if err := yaml.Unmarshal(data, obj); err != nil {
			return fmt.Errorf("error parsing %s: %v", source, err)
		}
		p.clusterRoleBindings = append(p.clusterRoleBindings, obj)
	default:
		return nil
	}

	return nil
}

func (p *FileStorage) sort() {
	sort.Slice(p.roles, func(i, j int) bool { return p.roles[i].Name < p.roles[j].Name })
	sort.Slice(p.roleBindings, func(i, j int) bool { return p.roleBindings[i].Name < p.roleBindings[j].Name })
	sort.Slice(p.clusterRoles, func(i, j int) bool { return p.clusterRoles[i].Name < p.clusterRoles[j].Name })
	sort.Slice(p.clusterRoleBindings, func(i, j int) bool { return p.clusterRoleBindings[i].Name < p.clusterRoleBindings[j].Name })
}
