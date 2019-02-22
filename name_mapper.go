package main

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/jinzhu/inflection"

	"github.com/iancoleman/strcase"
)

func init() {
	// add exceptions to the singularize all names rule
	inflection.AddSingular("annotations", "annotations")
	inflection.AddSingular("^(.*labels)$", "${1}")
	inflection.AddSingular("limits", "limits")
	inflection.AddSingular("metadata", "metadata")
	inflection.AddSingular("resources", "resources")
	inflection.AddSingular("requests", "requests")
}

// ToTerraformAttributeName takes the reflect.StructField data of a Kubernetes object attribute
// and translates it to the equivalent `terraform-kubernetes-provider` schema format.
//
// Sometimes the Kubernetes attribute name doesn't match struct name
//   e.g. `type ContainerPort struct` -> "ports" in YAML
// so we need to extract the JSON name from the StructField tag.
// Finally, the attribute name is converted to snake case.
func ToTerraformAttributeName(field reflect.StructField) string {
	name := extractJsonName(field)

	return normalizeTerraformName(name, false)
}

// ToTerraformSubBlockName takes the reflect.StructField data of a Kubernetes object attribute
// and translates it to the equivalent `terraform-kubernetes-provider` schema format.
//
// Sometimes the Kubernetes block name doesn't match struct name
//   e.g. `type ContainerPort struct` -> "ports" in YAML
// so we need to extract the JSON name from the StructField tag.
// Next, the attribute name is converted to singular + snake case.
func ToTerraformSubBlockName(field reflect.StructField) string {
	name := extractJsonName(field)

	return normalizeTerraformName(name, true)
}

// normalizeTerraformName converts the given string to snake case
// and optionally to singular form of the given word
func normalizeTerraformName(s string, toSingular bool) string {
	if toSingular {
		s = inflection.Singular(s)
	}
	s = strcase.ToSnake(s)
	return s
}

// extractJsonName inspects the StructField Tags to find the
// name used in JSON marshaling. This more accurately reflects
// the name expected by the API, and in turn the provider schema
func extractJsonName(field reflect.StructField) string {
	jsonTag := field.Tag.Get("json")
	if jsonTag == "" {
		fmt.Printf("WARNING - field [%s] has no json tag value", field.Name)
		return field.Name
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

func ToTerraformResourceType(v reflect.Value) string {
	ty := reflect.TypeOf(v.Interface())
	return "kubernetes_" + normalizeTerraformName(ty.Name(), true)
}

func ToTerraformResourceName(v reflect.Value) string {

	return "foo"
}
