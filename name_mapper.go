package main

import (
	"reflect"
	"strings"

	"github.com/jinzhu/inflection"

	"github.com/iancoleman/strcase"
)

func init() {
	inflection.AddSingular("metadata", "metadata")
}

// ToTerraformAttributeName takes the reflect.StructField data of a Kubernetes object attribute
// and translates it to the equivalent `terraform-kubernetes-provider` schema format.
//
// Sometimes the Kubernetes attribute name doesn't match struct name
//   e.g. `type ContainerPort struct` -> "ports" in YAML
// so we need to extract the JSON name from the StructField tag.
// Finally, the attribute name is converted to snake case.
func ToTerraformAttributeName(field reflect.StructField) string {
	name := extractJsonName(field.Tag)

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
	name := extractJsonName(field.Tag)

	return normalizeTerraformName(name, true)
}

func normalizeTerraformName(s string, toSingular bool) string {
	if toSingular {
		s = inflection.Singular(s)
	}
	s = strcase.ToSnake(s)
	return s
}

func extractJsonName(tag reflect.StructTag) string {
	jsonTag := tag.Get("json")
	if jsonTag == "" {
		panic("field has no json tag value")
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
