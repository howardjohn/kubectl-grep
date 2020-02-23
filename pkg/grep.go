package pkg

import "io"

type Resource struct {
	Name      string
	Namespace string
	Kind      string
}

func GrepResources(resources []Resource, in io.Reader) string {
	return ""
}
