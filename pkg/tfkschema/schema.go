package tfkschema

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-provider-kubernetes/kubernetes"
)

var ErrAttrNotFound = fmt.Errorf("could not find attribute in resource schema")

// ResourceSchema returns the named Terraform Provider Resource schema
// as defined in the `terraform-provider-kubernetes` package
func ResourceSchema(name string) *schema.Resource {
	prov := kubernetes.Provider()

	if res, ok := prov.ResourcesMap[name]; ok {
		return res
	}

	return nil
}

// ResourceField returns the Terraform schema object for the named resource field
// attrName should be in the form <resource>.path.to.field
func ResourceField(attrName string) *schema.Schema {
	if res, attrParts, ok := processAttributeName(attrName); ok {
		attr := search(res.Schema, attrParts)
		if attr != nil {
			return attr
		}
	}
	return nil
}

// IsKubernetesKindSupported returns true if a matching resource is found in the Terraform provider
func IsKubernetesKindSupported(obj runtime.Object) bool {
	name := ToTerraformResourceType(obj)

	res := ResourceSchema(name)
	if res != nil {
		return true
	}

	return false
}

// IsAttributeSupported scans the Terraform resource to determine if the named
// attribute is supported by the Kubernetes provider.
func IsAttributeSupported(attrName string) bool {
	if res, attrParts, ok := processAttributeName(attrName); ok {
		attr := search(res.Schema, attrParts)
		if attr != nil {
			return true
		}
	}
	return false
}

// IsAttributeRequired scans the Terraform resource to determine if the named
// attribute is required by the Kubernetes provider.
func IsAttributeRequired(attrName string) bool {
	if res, attrParts, ok := processAttributeName(attrName); ok {
		schemaMap := res.Schema

		attr := search(schemaMap, attrParts)
		if attr != nil {
			return attr.Required
		}
	}

	return false
}

func search(m map[string]*schema.Schema, attrParts []string) *schema.Schema {
	if len(attrParts) > 0 {
		searchKey := attrParts[0]
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}

		if v, ok := m[searchKey]; ok {
			if len(attrParts) == 1 {
				// we hit the bottom of our search and found the attribute
				return v
			}

			if v.Elem != nil {
				switch v.Elem.(type) {
				case *schema.Resource:
					res := v.Elem.(*schema.Resource)
					return search(res.Schema, attrParts[1:])
				}
			}

		}
	}

	return nil
}

// processAttributeName (naming things is hard) is a convenience method that splits a
// given resource attribute name, returning the identified Terraform resource (if any),
// and a slice of the remaining attribute path elements.
func processAttributeName(attr string) (*schema.Resource, []string, bool) {
	attrParts := strings.Split(attr, ".")
	res := ResourceSchema(attrParts[0])
	if res == nil {
		return nil, nil, false
	}

	return res, attrParts[1:], true
}
