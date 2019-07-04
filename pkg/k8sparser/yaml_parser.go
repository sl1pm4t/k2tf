package k8sparser

import (
	"bufio"
	"fmt"
	"io"

	multierror "github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog/log"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

func ParseYAML(in io.Reader) ([]runtime.Object, error) {
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
			log.Error().Err(err)
			result = multierror.Append(result, err)
		}
		d := scheme.Codecs.UniversalDeserializer()
		obj, _, err := d.Decode(doc, nil, nil)
		if err != nil {
			log.Error().Err(err)
			wrapped := fmt.Errorf("could not decode yaml object #%d: %s", i, err)
			result = multierror.Append(result, wrapped)
		}

		if obj != nil {
			objs = append(objs, obj)
		}
	}

	return objs, result
}

func ParseJSON(doc []byte) (runtime.Object, error) {
	var result error

	d := scheme.Codecs.UniversalDeserializer()
	obj, _, err := d.Decode(doc, nil, nil)
	if err != nil {
		wrapped := fmt.Errorf("could not decode JSON object: %s", err)
		result = multierror.Append(result, wrapped)
	}

	return obj, result
}
