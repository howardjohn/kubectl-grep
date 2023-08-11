package pkg

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestGrepResources(t *testing.T) {
	tests := []struct {
		name  string
		input string
		opts  Opts
		want  string
	}{
		{
			name:  "empty",
			input: "",
			want:  ``,
			opts:  Opts{Mode: Summary},
		},
		{
			name:  "list with newlines in configmap",
			opts:  Opts{Mode: Summary},
			input: readFile(t, "newlines.yaml"),
			want: `ConfigMap/cluster-autoscaler-status.kube-system
ConfigMap/istio-ca-root-cert.default
ConfigMap/kube-root-ca.crt.default
`,
		},
		{
			name:  "diff regression",
			input: readFile(t, "diff.json"),
			opts: Opts{
				Diff: true,
			},
			want: readFile(t, "diff.json.out"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := bytes.Buffer{}
			if err := GrepResources(tt.opts, strings.NewReader(tt.input), &o); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(o.String(), tt.want) {
				t.Errorf("got = %v, want %v", o.String(), tt.want)
			}
		})
	}
}

func readFile(t *testing.T, name string) string {
	res, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(res)
}
