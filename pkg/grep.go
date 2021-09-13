package pkg

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"
)

type Selector struct {
	Resources   []Resource
	Regex       *regexp.Regexp
	InvertRegex bool
}

func (s Selector) MatchesAll() bool {
	return len(s.Resources) == 0 && s.Regex == nil
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

func (o KubernetesObject) MatchesAny(sel Selector, text []byte) bool {
	if sel.Regex != nil {
		if sel.InvertRegex {
			if sel.Regex.Match(text) {
				return false
			}
		} else {
			if !sel.Regex.Match(text) {
				return false
			}
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

func GrepResources(sel Selector, in io.Reader, out io.Writer, mode DisplayMode) error {
	output := func(d string) {
		_, _ = fmt.Fprint(out, d)
	}
	r := bufio.NewReader(in)
	reader := NewYAMLReader(r)
	first := true
	for {
		text, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read document: %v", err)
		}
		// Optimization: Do not do YAML marshal if not needed
		if sel.MatchesAll() && mode == Full {
			if !first {
				fmt.Fprint(out, "---\n")
			}
			output(string(text))
			first = false
			continue
		}
		obj := KubernetesObject{}
		if err := yaml.Unmarshal(text, &obj); err != nil {
			return fmt.Errorf("failed to unmarshal yaml (%v): %v", string(text), err)
		}
		if obj.MatchesAny(sel, text) {
			if mode == Summary {
				if !obj.Empty() {
					output(obj.String() + "\n")
				}
			} else if mode == Clean || mode == CleanStatus {
				raw := genericMap{}
				if err := yaml.Unmarshal(text, &raw); err != nil {
					return err
				}
				o, err := yaml.Marshal(strip(raw, mode))
				if err != nil {
					return err
				}
				if len(raw) == 0 {
					o = []byte("")
				}
				if !first {
					fmt.Fprint(out, "---\n")
				}
				output(string(o))
			} else {
				if !first {
					fmt.Fprint(out, "---\n")
				}
				output(string(text))
			}
		}
		first = false
	}
	return nil
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
