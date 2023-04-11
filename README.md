# Kubectl Grep Plugin

A plugin to filter resources Kubernetes resources in YAML.
`kubectl grep` will preserve the original YAML structure, including comments, when transformations are not applied.

In addition to filtering out objects, a variety of transformations can be applied:
* `List` types will automatically be unrolled.
* `Secret` types can be decoded (base64) with `--decode/-d`.
* Noisy fields can be removed with `--clean/-n`, and status can additional be exluded with `--clean-status/-N`.
* The list of object names and types can be summarized with `--summary/-s`.

## Install

`go install github.com/howardjohn/kubectl-grep@latest`

## Usage

```shell
A plugin to grep Kubernetes resources.

Usage:
  kubectl-grep [flags]

Flags:
  -n, --clean               Cleanup generate fields
  -N, --clean-status        Cleanup generate fields, including status
  -d, --decode              Decode base64 fields in Secrets
  -w, --diff                Show diff of changes. Use with 'kubectl -ojson -w | kubectl grep -w'
      --diff-mode string    Format for diffs. Can be [line, inline]. (default "line")
  -h, --help                help for kubectl-grep
  -i, --insensitive-regex   Invert regex match
  -v, --invert-regex        Invert regex match
  -r, --regex string        Raw regex to match against
  -s, --summary             Summarize output
```

### Matching

`kubectl grep` takes matching clauses as arguments.
These follow the form `<KIND>/<NAME>.<NAMESPACE>`.
For example, `Service/kubernetes.default`.

Partial forms are also accepted:
* `<KIND>/<NAME>`
* `<KIND>/`
* `<NAME>`

Wildcards are allowed as well. For example:
* `*/name.namespace` - match any kinds
* `Service/*.default` - match all services in `default`
* `Service/echo-*.default` - match all services in `default` with a naming starting with `echo-`

### Grepping

Regex matches can be applied on a per-object basis.

For example, `kubectl grep -r security` will search all objects that have the phrase 'security' within them.

`-i` can make this case insensitive, and `-v` can invert the match.

### Diffing

A stream of events, as output by `kubectl get -w -ojson`, can be processed and show the diffs of what is changing.
This can be helpful to debug controllers.

For example:

```shell
kubectl get pods -w -ojson | kubectl grep -w
```

## Examples

#### Apply just the Services in some configuration

```shell
cat some-config.yaml | kubectl grep Service/ | kubectl apply -f -
```

#### Find a specific resource

```shell
cat some-config.yaml | kubectl grep Service/helloworld.default
```

#### Display all Pods in the `dev` namespace, hiding fields that add clutter like `managedFields`

```shell
cat some-config.yaml | kubectl grep 'Pod/*/dev' -N
```
#### Display all resources that contain the string `pertrytimeout` (case-insensitive), but do not contain `timeout`.

```shell
cat some-config.yaml | kubectl grep -r pertrytimeout -i | kubectl grep -v -r timeout
```
