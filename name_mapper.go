package main

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/jinzhu/inflection"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/iancoleman/strcase"
)

func init() {
	// add exceptions to the singularize all names rule
	inflection.AddSingular("annotations", "annotations")
	inflection.AddSingular("^(.*labels)$", "${1}")
	inflection.AddSingular("limits", "limits")
	inflection.AddSingular("resources", "resources")
	inflection.AddSingular("requests", "requests")

	inflection.AddUncountable("data")
	inflection.AddUncountable("metadata")
	inflection.AddUncountable("items")
}

// ToTerraformAttributeName takes the reflect.StructField data of a Kubernetes object attribute
// and translates it to the equivalent `terraform-kubernetes-provider` schema format.
//
// Sometimes the Kubernetes attribute name doesn't match struct name
//   e.g. `type ContainerPort struct` -> "ports" in YAML
// so we need to extract the JSON name from the StructField tag.
// Finally, the attribute name is converted to snake case.
func ToTerraformAttributeName(field *reflect.StructField, path string) string {
	name := extractProtobufName(field)

	return normalizeTerraformName(name, false, path)
}

// ToTerraformSubBlockName takes the reflect.StructField data of a Kubernetes object attribute
// and translates it to the equivalent `terraform-kubernetes-provider` schema format.
//
// Sometimes the Kubernetes block name doesn't match struct name
//   e.g. `type ContainerPort struct` -> "ports" in YAML
// so we need to extract the JSON name from the StructField tag.
// Next, the attribute name is converted to singular + snake case.
func ToTerraformSubBlockName(field *reflect.StructField, path string) string {
	name := extractProtobufName(field)

	return normalizeTerraformName(name, true, path)
}

// normalizeTerraformName converts the given string to snake case
// and optionally to singular form of the given word
// s is the string to normalize
// set toSingular to true to singularize the given word
// path is the full schema path to the named element
func normalizeTerraformName(s string, toSingular bool, path string) string {
	switch s {
	case "DaemonSet":
		return "daemonset"

	case "nonResourceURLs":
		if strings.Contains(path, "role.rule.") {
			return "non_resource_urls"
		}

	case "updateStrategy":
		if !strings.Contains(path, "stateful") {
			return "strategy"
		}
	}

	if toSingular {
		s = inflection.Singular(s)
	}
	s = strcase.ToSnake(s)

	return s
}

// extractJsonName inspects the StructField Tags to find the
// name used in JSON marshaling. This more accurately reflects
// the name expected by the API, and in turn the provider schema
func extractJsonName(field *reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		fmt.Fprintf(os.Stderr, "WARNING - field [%s] has no json tag value\n", field.Name)
		return extractProtobufName(field)
	}

	comma := strings.Index(jsonTag, ",")
	var name string
	if comma != -1 {
		name = jsonTag[:comma]
	} else {
		name = jsonTag
	}

	return name
}

func extractProtobufName(field *reflect.StructField) string {
	protoTag := field.Tag.Get("protobuf")
	if protoTag == "" {
		fmt.Fprintf(os.Stderr, "WARNING - field [%s] has no protobuf tag\n", field.Name)
		return field.Name
	}

	tagParts := strings.Split(protoTag, ",")
	for _, part := range tagParts {
		if strings.Contains(part, "name=") {
			return part[5:]
		}
	}

	fmt.Fprintf(os.Stderr, "WARNING - field [%s] protobuf tag has no 'name'\n", field.Name)
	return field.Name
}

// ToTerraformResourceType converts a Kubernetes API Object Type name to the
// equivalent `terraform-provider-kubernetes` schema name.
func ToTerraformResourceType(obj runtime.Object) string {
	tmeta := typeMeta(obj)

	return "kubernetes_" + normalizeTerraformName(tmeta.Kind, false, "")
}

// ToTerraformResourceName extract the Kubernetes API Objects' name from the
// ObjectMeta
func ToTerraformResourceName(obj runtime.Object) string {
	meta := objectMeta(obj)

	return normalizeTerraformName(meta.Name, false, "")
}

// NormalizeTerraformMapKey converts Map keys to a form suitable for Terraform
// HCL
//
// e.g. map keys that include certain characters ( '/' ) will be wrapped in
// double quotes.
func NormalizeTerraformMapKey(s string) string {
	if strings.Contains(s, "/") {
		return fmt.Sprintf(`"%s"`, s)
	}
	return s
}

func objectMeta(obj runtime.Object) metav1.ObjectMeta {
	v := reflect.ValueOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	metaF := v.FieldByName("ObjectMeta")

	return metaF.Interface().(metav1.ObjectMeta)
}

func typeMeta(obj runtime.Object) metav1.TypeMeta {
	v := reflect.ValueOf(obj)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	metaF := v.FieldByName("TypeMeta")

	return metaF.Interface().(metav1.TypeMeta)
}
