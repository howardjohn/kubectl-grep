package pkg

import (
	"io"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v2"
)

type Resource struct {
	Name      string
	Namespace string
	Kind      string
}

type KubernetesObject struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"metadata"`
}

type KubernetesListRaw struct {
	Items []map[interface{}]interface{} `json:"items"`
}

type KubernetesListMeta struct {
	Items []KubernetesObject `json:"items"`
}

func match(pattern string, s string) bool {
	if pattern == "" || pattern == "*" {
		return true
	}
	if s == "" {
		return false
	}
	if strings.HasPrefix(pattern, "*") {
		return strings.HasSuffix(s, pattern[1:])
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(s, pattern[:len(pattern)-1])
	}
	return pattern == s
}

func (o KubernetesObject) Matches(r Resource) bool {
	if !match(r.Kind, o.Kind) {
		return false
	}
	if !match(r.Name, o.Metadata.Name) {
		return false
	}
	if !match(r.Namespace, o.Metadata.Namespace) {
		return false
	}
	return true
}

func (o KubernetesObject) MatchesAny(rs []Resource) bool {
	if len(rs) == 0 {
		return true
	}
	for _, r := range rs {
		if o.Matches(r) {
			return true
		}
	}
	return false
}

func GrepResources(resources []Resource, in io.Reader) (string, error) {
	st, err := ioutil.ReadAll(in)
	if err != nil {
		return "", err
	}
	matches := []string{}
	for _, text := range strings.Split(string(st), "\n---") {
		obj := KubernetesObject{}
		if err := yaml.Unmarshal([]byte(text), &obj); err != nil {
			return "", err
		}
		if obj.Kind == "List" {
			objs, metas, err := decomposeList(text)
			if err != nil {
				return "", err
			}
			for i, raw := range objs {
				meta := metas[i]
				if meta.MatchesAny(resources) {
					o, err := yaml.Marshal(raw)
					if err != nil {
						return "", err
					}
					matches = append(matches, "\n"+string(o))
				}
			}
		} else {
			if obj.MatchesAny(resources) {
				matches = append(matches, text)
			}
		}
	}
	return strings.Join(matches, "\n---"), nil
}

func decomposeList(text string) ([]map[interface{}]interface{}, []KubernetesObject, error) {
	m := KubernetesListMeta{}
	if err := yaml.Unmarshal([]byte(text), &m); err != nil {
		return nil, nil, err
	}
	r := KubernetesListRaw{}
	if err := yaml.Unmarshal([]byte(text), &r); err != nil {
		return nil, nil, err
	}
	return r.Items, m.Items, nil
}
