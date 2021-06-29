package pkg

import (
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type Selector struct {
	Resources []Resource
	Regex     *regexp.Regexp
}

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

func (o KubernetesObject) String() string {
	return fmt.Sprintf("%s/%s.%s", o.Kind, o.Metadata.Name, o.Metadata.Namespace)
}

type genericMap = map[interface{}]interface{}

type KubernetesListRaw struct {
	Items []genericMap `json:"items"`
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

func (o KubernetesObject) MatchesAny(sel Selector, text string) bool {
	if sel.Regex != nil {
		if !sel.Regex.MatchString(text) {
			return false
		}
	}
	if len(sel.Resources) == 0 {
		return true
	}
	for _, r := range sel.Resources {
		if o.Matches(r) {
			return true
		}
	}
	return false
}

func (o KubernetesObject) Empty() bool {
	return o.Kind == "" && o.Metadata.Name == "" && o.Metadata.Namespace == ""
}

type DisplayMode int

const (
	Full DisplayMode = iota
	Summary
	Clean
	CleanStatus
)

func GrepResources(sel Selector, in io.Reader, mode DisplayMode) (string, error) {
	st, err := ioutil.ReadAll(in)
	if err != nil {
		return "", err
	}
	matches := []string{}
	for _, text := range strings.Split(string(st), "\n---") {
		obj := KubernetesObject{}
		if err := yaml.Unmarshal([]byte(text), &obj); err != nil {
			return "", fmt.Errorf("failed to unmarshal yaml (%v): %v", text, err)
		}
		if obj.Kind == "List" {
			objs, metas, err := decomposeList(text)
			if err != nil {
				return "", err
			}
			for i, raw := range objs {
				meta := metas[i]
				txt := ""
				if sel.Regex != nil {
					got, err := yaml.Marshal(raw)
					if err != nil {
						panic(err.Error())
					}
					txt = string(got)
				}
				if meta.MatchesAny(sel, txt) {
					o, err := yaml.Marshal(strip(raw, mode))
					if err != nil {
						return "", err
					}
					if mode == Summary {
						matches = append(matches, meta.String())
					} else {
						matches = append(matches, "\n"+string(o))
					}
				}
			}
		} else {
			if obj.MatchesAny(sel, text) {
				if mode == Summary {
					if !obj.Empty() {
						matches = append(matches, obj.String())
					}
				} else if mode == Clean || mode == CleanStatus {
					raw := genericMap{}
					if err := yaml.Unmarshal([]byte(text), &raw); err != nil {
						return "", err
					}
					o, err := yaml.Marshal(strip(raw, mode))
					if err != nil {
						return "", err
					}
					matches = append(matches, "\n"+string(o))
				} else {
					matches = append(matches, text)
				}
			}
		}
	}
	if mode == Summary {
		return strings.Join(matches, "\n"), nil
	}
	return strings.Join(matches, "\n---"), nil
}

func strip(raw genericMap, mode DisplayMode) interface{} {
	if mode == Clean || mode == CleanStatus {
		deleteNested(raw, "metadata", "annotations", "kubectl.kubernetes.io/last-applied-configuration")
		deleteNested(raw, "metadata", "generation")
		deleteNested(raw, "metadata", "resourceVersion")
		deleteNested(raw, "metadata", "selfLink")
		deleteNested(raw, "metadata", "uid")
		deleteNested(raw, "metadata", "creationTimestamp")
		deleteNested(raw, "metadata", "generateName")
		deleteNested(raw, "metadata", "ownerReferences")
		deleteNested(raw, "metadata", "managedFields")
		deleteNested(raw, "metadata", "labels", "pod-template-hash")
	}
	if mode == CleanStatus {
		deleteNested(raw, "status")
	}
	return raw
}

func deleteNested(raw genericMap, keys ...string) {
	if len(keys) == 0 {
		return
	}
	if len(keys) == 1 {
		delete(raw, keys[0])
	}
	meta, ok := raw[keys[0]]
	if ok {
		metamap, ok := meta.(genericMap)
		if ok {
			deleteNested(metamap, keys[1:]...)
			if len(metamap) == 0 {
				delete(raw, keys[0])
			}
		}
	}
}

func decomposeList(text string) ([]genericMap, []KubernetesObject, error) {
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
