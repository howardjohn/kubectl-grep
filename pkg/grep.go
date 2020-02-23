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

func (o KubernetesObject) Matches(r Resource) bool {
	if r.Kind != "" && o.Kind != r.Kind {
		return false
	}
	if r.Name != "" && o.Metadata.Name != r.Name {
		return false
	}
	if r.Namespace != "" && o.Metadata.Namespace != r.Namespace {
		return false
	}
	return true
}

func (o KubernetesObject) MatchesAny(rs []Resource) bool {
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
		if obj.MatchesAny(resources) {
			matches = append(matches, text)
		}
	}
	return strings.Join(matches, "\n---"), nil
}
