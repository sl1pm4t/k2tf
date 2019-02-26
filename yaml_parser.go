package main

import (
	"bufio"
	"fmt"
	"io"

	multierror "github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func ParseK8SYAML(in io.Reader) ([]runtime.Object, error) {
	var result error
	objs := []runtime.Object{}

	b := bufio.NewReader(in)
	r := yaml.NewYAMLReader(b)

	for i := 1; ; i++ {
		doc, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result = multierror.Append(result, err)
		}
		d := scheme.Codecs.UniversalDeserializer()
		obj, _, err := d.Decode(doc, nil, nil)
		if err != nil {
			wrapped := fmt.Errorf("could not decode yaml object #%d: %s", i, err)
			result = multierror.Append(result, wrapped)
		}

		objs = append(objs, obj)
	}

	return objs, result
}

func ParseK8SJSON(doc []byte) (runtime.Object, error) {
	var result error

	d := scheme.Codecs.UniversalDeserializer()
	obj, _, err := d.Decode(doc, nil, nil)
	if err != nil {
		wrapped := fmt.Errorf("could not decode JSON object: %s", err)
		result = multierror.Append(result, wrapped)
	}

	return obj, result
}
