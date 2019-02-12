package main

import (
	"fmt"
	"reflect"

	spew "github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/zclconf/go-cty/cty"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var KubeKindTFResourceTypeMap = map[string]string{
	"Deployment": "kubernetes_deployment",
	"ConfigMap":  "kubernetes_config_map",
}

func WriteObject(obj runtime.Object, dst *hclwrite.Body) {
	kubeKind := obj.GetObjectKind().GroupVersionKind().Kind
	kubeVersion := obj.GetObjectKind().GroupVersionKind().Version

	switch {
	case kubeKind == "ConfigMap":
		// WriteConfigMap(obj.(*corev1.ConfigMap), dst)

	case kubeKind == "Deployment" && kubeVersion == "v1":
		// WriteDeployment(obj.(*v1.Deployment), dst)

	default:
		printFields(obj, "")
	}
}

func printFields(obj interface{}, indent string) {
	fmt.Printf("%s---\n", indent)
	rv := reflect.ValueOf(obj)
	ty := rv.Type()
	if ty.Kind() == reflect.Ptr {
		rv = rv.Elem()
		ty = rv.Type()
	}

	fmt.Printf("%sType %s\n", indent, ty)
	fmt.Printf("%sKind %s\n", indent, ty.Kind())
	fmt.Printf("%sNumber of fields %d\n", indent, rv.NumField())
	indent = indent + "  "
	for i := 0; i < rv.NumField(); i++ {
		fmt.Printf("%s---\n", indent)
		fieldName := ty.Field(i).Name
		if ignoredField(fieldName) {
			continue
		}

		field := ty.Field(i)
		fieldTy := field.Type
		fieldVal := rv.Field(i)
		fmt.Printf("%sField Name: %s\n", indent, ty.Field(i).Name)
		fmt.Printf("%sField Type: %s\n", indent, fieldVal.Type())
		fmt.Printf("%sField Kind: %s\n", indent, fieldVal.Kind())
		fmt.Printf("%sField Tags: %s\n", indent, ty.Field(i).Tag)

		if fieldVal.Kind() == reflect.Struct {
			printFields(fieldVal.Interface(), indent+"  ")

		} else if fieldVal.Kind() == reflect.Slice {
			fmt.Printf("%sField Value: %v\n", indent, fieldVal)
			fmt.Printf("%sSLICE: %s -- %s\n", indent, spew.Sdump(fieldVal), spew.Sdump(fieldVal.Interface()))

			s := fieldVal.Interface()
			typ := reflect.TypeOf(s).Elem()
			fmt.Printf("%sSlice Elem Type: %v\n", indent, typ)

		} else {
			if fieldTy.Kind() == reflect.Ptr {
				fieldTy = fieldTy.Elem()
				fieldVal = fieldVal.Elem()
			}
			if !fieldVal.IsValid() {
				continue
			}
			fmt.Printf("%sField Value: %v\n", indent, fieldVal)

		}

		// fmt.Printf(s%spew.Sdump(fieldTy))
		// fmt.Printf(s%spew.Sdump(fieldVal))

		// if fieldTy.Kind() == reflect.Ptr {
		// 	fieldTy = fieldTy.Elem()
		// 	fieldVal = fieldVal.Elem()
		// }

		// fmt.Printf("Field:%d Name:%s type:%T value:%v\n", i, ty.Field(i).Name, field.Type, fieldVal)
		// fmt.Printf(s%spew.Sdump(field))
		// fmt.Printf(s%spew.Sdump(fieldTy))
		// fmt.Printf(s%spew.Sdump(fieldVal))
	}
}

// func getObjectFields(obj runtime.Object) {
// 	rv := reflect.ValueOf(val)
// 	ty := rv.Type()
// 	if ty.Kind() == reflect.Ptr {
// 		rv = rv.Elem()
// 		ty = rv.Type()
// 	}
// 	if ty.Kind() != reflect.Struct {
// 		panic(fmt.Sprintf("value is %s, not struct", ty.Kind()))
// 	}

// 	tags := getFieldTags(ty)

// }

func encodeMetadataBlock(meta *metav1.ObjectMeta) *hclwrite.Block {
	blk := hclwrite.NewBlock("metadata", nil)
	blk.Body().SetAttributeValue("name", convertCtyValue(meta.Name))

	if meta.Namespace != "" {
		blk.Body().SetAttributeValue("namespace", convertCtyValue(meta.Namespace))
	}

	if len(meta.Labels) > 0 {
		lbls := hclwrite.NewBlock("labels", nil)
		encodeMap(meta.Labels, lbls.Body())
		blk.Body().AppendBlock(lbls)
	}

	if len(meta.Annotations) > 0 {
		anno := hclwrite.NewBlock("annotations", nil)
		encodeMap(meta.Annotations, anno.Body())
		blk.Body().AppendBlock(anno)
	}
	return blk
}

func WriteConfigMap(obj *corev1.ConfigMap, dst *hclwrite.Body) {
	root := hclwrite.NewBlock("resource", []string{"kubernetes_config_map", obj.Name})
	dst.AppendBlock(root)

	root.Body().AppendBlock(encodeMetadataBlock(&obj.ObjectMeta))

	if len(obj.Data) > 0 {
		data := hclwrite.NewBlock("data", nil)
		encodeMap(obj.Data, data.Body())
		root.Body().AppendBlock(data)
	}

}

func WriteDeployment(obj *v1.Deployment, dst *hclwrite.Body) {
	root := hclwrite.NewBlock("resource", []string{"kubernetes_deployment", obj.Name})
	dst.AppendBlock(root)

	root.Body().AppendBlock(encodeMetadataBlock(&obj.ObjectMeta))

	spec := hclwrite.NewBlock("spec", nil)
	spec.Body().SetAttributeValue("replicas", convertCtyValue(obj.Spec.Replicas))

	root.Body().AppendBlock(spec)
}

func convertCtyValue(val interface{}) cty.Value {
	switch val.(type) {
	case string:
		return cty.StringVal(val.(string))
	case bool:
		return cty.BoolVal(val.(bool))
	case int:
		return cty.NumberIntVal(int64(val.(int)))
	case int32:
		return cty.NumberIntVal(int64(val.(int32)))
	case *int32:
		val = *val.(*int32)
		return cty.NumberIntVal(int64(val.(int32)))
	case int64:
		return cty.NumberIntVal(int64(val.(int64)))
	case float64:
		return cty.NumberFloatVal(float64(val.(float64)))
	case map[string]interface{}:
		ctyMap := map[string]cty.Value{}
		for k, v := range val.(map[string]interface{}) {
			ctyMap[k] = convertCtyValue(v)
		}

		return cty.ObjectVal(ctyMap)
	default:
		fmt.Printf("[!] WARN: unhandled variable type: %T \n", val)
	}
	return cty.NilVal
}

func encodeMap(m map[string]string, dst *hclwrite.Body) {
	for k, v := range m {
		dst.SetAttributeValue(k, convertCtyValue(v))
	}
}

var ignoredFields = []string{
	"CreationTimestamp",
	"DeletionTimestamp",
	"Generation",
	"SelfLink",
	"UID",
	"ResourceVersion",
	"TypeMeta",
	"Status",
}
var ignoredFieldMap map[string]bool

func init() {
	ignoredFieldMap = make(map[string]bool, len(ignoredFields))
	for _, v := range ignoredFields {
		ignoredFieldMap[v] = true
	}
}

func ignoredField(name string) bool {
	_, ok := ignoredFieldMap[name]
	return ok
}
