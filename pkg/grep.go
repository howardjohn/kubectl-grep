package pkg

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
	"github.com/sters/yaml-diff/yamldiff"
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

type DiffType int

const (
	DiffLine   DiffType = iota
	DiffInline DiffType = iota
)

type Differ struct {
	mode DiffType
	objs map[KubernetesObject]string
}

func (d *Differ) Add(obj KubernetesObject, now string) string {
	old, f := d.objs[obj]
	d.objs[obj] = now
	if f {
		if d.mode == DiffLine {
			oy, _ := yamldiff.Load(old)
			ny, _ := yamldiff.Load(now)
			d := yamldiff.Do(oy, ny)
			return d[0].Dump()
		} else {
			dmp := diffmatchpatch.New()
			diffs := dmp.DiffMain(old, now, false)
			return dmp.DiffPrettyText(dmp.DiffCleanupSemanticLossless(diffs))
		}
	} else {
		return now
	}
}

type Opts struct {
	Sel      Selector
	Mode     DisplayMode
	Diff     bool
	DiffType DiffType
	Decode   bool
}

func GrepResources(opts Opts, in io.Reader, out io.Writer) error {
	r := bufio.NewReader(in)
	reader := NewYAMLReader(r)
	first := true
	differ := &Differ{
		mode: opts.DiffType,
		objs: map[KubernetesObject]string{},
	}
	output := func(obj KubernetesObject, d string) {
		if !first && opts.Mode != Summary {
			_, _ = fmt.Fprint(out, "---\n")
		}
		first = false
		if opts.Diff {
			d = differ.Add(obj, d)
		}
		_, _ = fmt.Fprint(out, d)
	}
	for {
		text, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read document: %v", err)
		}
		obj := KubernetesObject{}
		// Optimization: Do not do YAML marshal if not needed
		if opts.Sel.MatchesAll() && opts.Mode == Full && !opts.Decode && !opts.Diff {
			output(obj, string(text))
			continue
		}
		if err := yaml.Unmarshal(text, &obj); err != nil {
			return fmt.Errorf("failed to unmarshal yaml (%v): %v", string(text), err)
		}
		if obj.MatchesAny(opts.Sel, text) {
			if opts.Mode == Summary {
				if !obj.Empty() {
					output(obj, obj.String()+"\n")
				}
			} else if opts.Mode == Clean || opts.Mode == CleanStatus || opts.Decode || opts.Diff {
				raw := genericMap{}
				if err := yaml.Unmarshal(text, &raw); err != nil {
					return err
				}
				raw = strip(raw, opts.Mode)
				if opts.Decode && obj.Kind == "Secret" {
					raw = decodeSecret(raw)
				}
				if opts.Decode && obj.Kind == "ConfigMap" {
					raw = decodeConfigMap(raw)
				}
				o, err := yaml.Marshal(raw)
				if err != nil {
					return err
				}
				if len(raw) == 0 {
					o = []byte("")
				}
				output(obj, string(o))
			} else {
				output(obj, string(text))
			}
		}
	}
	return nil
}

func decodeSecret(raw genericMap) genericMap {
	data, ok := raw["data"]
	if !ok {
		return raw
	}
	gm, ok := data.(genericMap)
	if !ok {
		return raw
	}
	for k, v := range gm {
		gm[k] = base64Decode(v)
	}
	return raw
}

func decodeConfigMap(raw genericMap) genericMap {
	data, ok := raw["binaryData"]
	if !ok {
		return raw
	}
	gm, ok := data.(genericMap)
	if !ok {
		return raw
	}
	for k, v := range gm {
		gm[k] = base64Decode(v)
	}
	return raw
}

func base64Decode(d interface{}) interface{} {
	t, ok := d.(string)
	if !ok {
		return d
	}
	b, err := base64.StdEncoding.DecodeString(t)
	if err != nil {
		return d
	}
	return string(b)
}

func strip(raw genericMap, mode DisplayMode) genericMap {
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
