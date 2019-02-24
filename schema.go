package main

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/terraform-providers/terraform-provider-kubernetes/kubernetes"
)

func ResourceSchema(name string) *schema.Resource {
	prov := kubernetes.Provider().(*schema.Provider)

	if res, ok := prov.ResourcesMap[name]; ok {
		return res
	}

	return nil
}

func SchemaSupportsAttribute(resName, attrName string) (bool, error) {
	res := ResourceSchema(resName)

	if res != nil {
		attrParts := strings.Split(attrName, ".")
		schemaMap := res.Schema

		return search(schemaMap, attrParts, 0)
	}

	return false, fmt.Errorf("could not find resource: %s", resName)
}

func search(m map[string]*schema.Schema, attrParts []string, depth int) (bool, error) {
	searchKey := attrParts[depth]
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// fmt.Printf("search -> depth=%d, searchKey=%s, keys=%v\n", depth, searchKey, keys)

	if v, ok := m[searchKey]; ok {

		if depth == len(attrParts)-1 {
			return true, nil
		}

		if v.Elem != nil {
			switch v.Elem.(type) {
			case *schema.Resource:
				res := v.Elem.(*schema.Resource)
				return search(res.Schema, attrParts, depth+1)
			}
		}

	}

	return false, fmt.Errorf("could not find attribute <%v> in resource [depth=%d]", attrParts, depth)
}
