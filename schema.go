package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-kubernetes/kubernetes"
)

var errAttrNotFound = fmt.Errorf("could not find attribute in resource schema")

// ResourceSchema returns the named Terraform Provider Resource schema
// as defined in the `terraform-provider-kubernetes` package
func ResourceSchema(name string) *schema.Resource {
	prov := kubernetes.Provider().(*schema.Provider)

	if res, ok := prov.ResourcesMap[name]; ok {
		return res
	}

	return nil
}

// IsAttributeSupported scans the Terraform resource to determine if the named
// attribute is supported by the Kubernetes provider.
func IsAttributeSupported(attrName string) (bool, error) {
	attrParts := strings.Split(attrName, ".")
	res := ResourceSchema(attrParts[0])
	if res == nil {
		return false, fmt.Errorf("could not find resource: %s", attrParts[0])
	}
	schemaMap := res.Schema

	return search(schemaMap, attrParts[1:])
}

func search(m map[string]*schema.Schema, attrParts []string) (bool, error) {
	searchKey := attrParts[0]
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	if v, ok := m[searchKey]; ok {
		if len(attrParts) == 1 {
			// we hit the bottom of our search and found the attribute
			return true, nil
		}

		if v.Elem != nil {
			switch v.Elem.(type) {
			case *schema.Resource:
				res := v.Elem.(*schema.Resource)
				return search(res.Schema, attrParts[1:])
			}
		}

	}

	return false, errAttrNotFound
}
