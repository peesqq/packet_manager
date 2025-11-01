package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v3"
)

type SSHConfig struct {
	Host       string `json:"host" yaml:"host"`
	Port       int    `json:"port" yaml:"port"`
	User       string `json:"user" yaml:"user"`
	Password   string `json:"password" yaml:"password"`
	RemotePath string `json:"remote_path" yaml:"remote_path"`
}

type Target struct {
	Path    string `json:"path" yaml:"path"`
	Exclude string `json:"exclude" yaml:"exclude"`
}

type Packet struct {
	Name    string     `json:"name" yaml:"name"`
	Ver     string     `json:"ver" yaml:"ver"`
	Targets []Target   `json:"targets" yaml:"targets"`
	SSH     *SSHConfig `json:"ssh" yaml:"ssh"`
	Output  string     `json:"output" yaml:"output"`
}

type PackageSpec struct {
	Name string `json:"name" yaml:"name"`
	Ver  string `json:"ver" yaml:"ver"`
}

type Packages struct {
	Packages  []PackageSpec `json:"packages" yaml:"packages"`
	SSH       *SSHConfig    `json:"ssh" yaml:"ssh"`
	OutputDir string        `json:"output_dir" yaml:"output_dir"`
}

type RemoteArchive struct {
	Name       string
	Version    string
	RemotePath string
}

func LoadPacket(path string) (*Packet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Packet
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, err
		}
	default:
		if json.Unmarshal(data, &p) == nil {
			return &p, nil
		}
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, err
		}
	}
	return &p, nil
}

func LoadPackages(path string) (*Packages, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p Packages
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, err
		}
	default:
		if json.Unmarshal(data, &p) == nil {
			return &p, nil
		}
		if err := yaml.Unmarshal(data, &p); err != nil {
			return nil, err
		}
	}
	return &p, nil
}

func (p *Packet) CollectFiles() ([]string, error) {
	var out []string
	for _, t := range p.Targets {
		matches, err := filepath.Glob(t.Path)
		if err != nil {
			return nil, err
		}
		for _, f := range matches {
			info, err := os.Stat(f)
			if err != nil {
				return nil, err
			}
			if info.IsDir() {
				continue
			}
			if t.Exclude != "" {
				if ok, _ := filepath.Match(t.Exclude, filepath.Base(f)); ok {
					continue
				}
			}
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no files found")
	}
	return out, nil
}

func ResolvePackages(remoteFiles []string, specs []PackageSpec) ([]RemoteArchive, error) {
	m := map[string][]RemoteArchive{}
	for _, r := range remoteFiles {
		bn := filepath.Base(r)
		if !strings.HasSuffix(bn, ".tar.gz") {
			continue
		}
		name, ver, ok := splitNameVersion(bn)
		if !ok {
			continue
		}
		m[name] = append(m[name], RemoteArchive{Name: name, Version: ver, RemotePath: r})
	}
	var result []RemoteArchive
	for _, s := range specs {
		cset, err := parseConstraint(s.Ver)
		if err != nil {
			return nil, err
		}
		candidates := m[s.Name]
		if len(candidates) == 0 {
			return nil, errors.New("package not found: " + s.Name)
		}
		bestIdx := -1
		var bestVer *semver.Version
		for i, c := range candidates {
			v, err := semver.NewVersion(c.Version)
			if err != nil {
				continue
			}
			if cset != nil && !cset.Check(v) {
				continue
			}
			if bestVer == nil || v.GreaterThan(bestVer) {
				bestVer = v
				bestIdx = i
			}
		}
		if bestIdx < 0 {
			return nil, errors.New("no version satisfies constraint for " + s.Name)
		}
		result = append(result, candidates[bestIdx])
	}
	return result, nil
}

func parseConstraint(s string) (*semver.Constraints, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	c, err := semver.NewConstraint(s)
	if err == nil {
		return c, nil
	}
	if strings.HasPrefix(s, "<=") || strings.HasPrefix(s, ">=") || strings.HasPrefix(s, "<") || strings.HasPrefix(s, ">") || strings.HasPrefix(s, "=") {
		return nil, err
	}
	c2, err2 := semver.NewConstraint("=" + s)
	if err2 == nil {
		return c2, nil
	}
	return nil, err
}

func splitNameVersion(filename string) (string, string, bool) {
	base := strings.TrimSuffix(filename, ".tar.gz")
	i := strings.LastIndex(base, "-")
	if i <= 0 {
		return "", "", false
	}
	name := base[:i]
	ver := base[i+1:]
	return name, ver, true
}
