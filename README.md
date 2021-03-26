# Kubectl Grep Plugin

A plugin to grep Kubernetes resources in YAML.
Unless transformations are applied, the original YAML structure, including comments, will be preserved.

`List` types will automatically be unrolled.

## Install

`go get github.com/howardjohn/kubectl-grep`

## Usage

```shell
A plugin to grep Kubernetes resources.

Usage:
  kubectl-grep [flags]

Flags:
  -n, --clean          Cleanup generate fields
  -N, --clean-status   Cleanup generate fields, including status
  -h, --help           help for kubectl-grep
  -s, --summary        Summarize output
  -L, --unlist         Split Kubernetes lists
```

## Examples

#### Apply just the Services in some configuration

```shell
< some-config.yaml | kubectl grep Service/ | kubectl apply -f -
```

#### Find a specific resource

```shell
< some-config.yaml | kubectl grep Service/helloworld.default
```

#### Display all Pods in the `dev` namespace, hiding fields that add clutter like `managedFields`

```shell
< some-config.yaml | kubectl grep 'Pod/*/dev' -N
```
