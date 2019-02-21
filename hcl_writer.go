package main

import (
	"fmt"
	"reflect"
	"strings"

	spew "github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/iancoleman/strcase"
	"github.com/mitchellh/reflectwalk"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var KubeKindTFResourceTypeMap = map[string]string{
	"Deployment": "kubernetes_deployment",
	"ConfigMap":  "kubernetes_config_map",
}

type walker struct {
	indent       string
	currentField string
	isTopLevel   bool

	dst      *hclwrite.Body
	typeName string
}

func NewWalker(dst *hclwrite.Body) *walker {
	w := &walker{}
	w.dst = dst
	w.isTopLevel = true

	return w
}

func (w *walker) Enter(l reflectwalk.Location) error {
	if l == reflectwalk.WalkLoc {
		w.isTopLevel = true
	} else {
		w.isTopLevel = false
	}

	if l == reflectwalk.Struct || l == reflectwalk.Slice || l == reflectwalk.Map {
		// fmt.Printf("%sentered %s\n", w.indent, l.String())
		w.indent = w.indent + "  "
	}
	return nil
}

func (w *walker) Exit(l reflectwalk.Location) error {
	if l == reflectwalk.Struct || l == reflectwalk.Slice || l == reflectwalk.Map {
		w.indent = w.indent[0 : len(w.indent)-2]
		fmt.Printf("%sexited %s\n", w.indent, l.String())
	}
	return nil
}

func (w *walker) Struct(v reflect.Value) error {
	ty := reflect.TypeOf(v.Interface())
	w.indent = w.indent + "  "

	fmt.Printf("%sEntered Struct: %s\n", w.indent, ty.Name())

	if w.isTopLevel {
		typeName := KubeKindTFResourceTypeMap[ty.Name()]

		// top level resource block
		resourceBlock := hclwrite.NewBlock("resource", []string{typeName, "blah"})
		w.dst.AppendBlock(resourceBlock)
	}

	return nil
}

func (w *walker) StructField(field reflect.StructField, v reflect.Value) error {
	if field.Anonymous {
		return reflectwalk.SkipEntry
	} else if !v.IsValid() {
		return reflectwalk.SkipEntry
	} else {
		switch field.Name {
		// case "ObjectMeta":
		// 	objMeta := fieldVal.Interface().(metav1.ObjectMeta)
		// 	metaBlock = encodeMetadataBlock(&objMeta)
		// 	resourceName = strcase.ToSnake(objMeta.Name)
		case "TypeMeta":
			typeMeta := v.Interface().(metav1.TypeMeta)
			w.typeName = "kubernetes_" + strcase.ToSnake(typeMeta.Kind)
		case "Status":
			// skip
		case "BinaryData":
			// skip
		default:
			w.currentField = field.Name
		}

	}
	return nil
}

func (w *walker) Primitive(v reflect.Value) error {
	if v.CanAddr() && v.CanInterface() {
		fmt.Printf("%sPrimitive: %s = %v\n", w.indent, w.currentField, v.Interface())
	}
	return nil
}

func (w *walker) Map(m reflect.Value) error {
	fmt.Printf("%sMap [%s]: \n", w.indent, w.currentField)
	return nil
}

func (w *walker) MapElem(m, k, v reflect.Value) error {
	fmt.Printf("%sMapElem: \n", w.indent)
	return nil
}

func WriteObject(obj runtime.Object, dst *hclwrite.Body) {
	w := NewWalker(dst)
	reflectwalk.Walk(obj, w)

	return

	kubeKind := obj.GetObjectKind().GroupVersionKind().Kind
	kubeVersion := obj.GetObjectKind().GroupVersionKind().Version

	switch {
	case kubeKind == "ConfigMap":
		// WriteConfigMap(obj.(*corev1.ConfigMap), dst)

	case kubeKind == "Deployment" && kubeVersion == "v1":
		// WriteDeployment(obj.(*v1.Deployment), dst)

	default:
		// printFields(obj, "")
	}

	rv := reflect.ValueOf(obj)
	ty := rv.Type()
	if ty.Kind() == reflect.Ptr {
		rv = rv.Elem()
		ty = rv.Type()
	}

	var typeName string
	var resourceName string
	var metaBlock *hclwrite.Block
	var otherBlocks []*hclwrite.Block
	for i := 0; i < rv.NumField(); i++ {
		fieldVal := rv.Field(i)
		fieldName := ty.Field(i).Name
		switch fieldName {
		case "ObjectMeta":
			objMeta := fieldVal.Interface().(metav1.ObjectMeta)
			metaBlock = encodeMetadataBlock(&objMeta)
			resourceName = strcase.ToSnake(objMeta.Name)
		case "TypeMeta":
			typeMeta := fieldVal.Interface().(metav1.TypeMeta)
			typeName = "kubernetes_" + strcase.ToSnake(typeMeta.Kind)
		case "Status":
			continue
		// case "BinaryData":
		// 	// don't add to TF resource
		// 	continue
		default:
			// must some other field like 'spec' or 'data'
			blk := EncodeAsBlock(fieldVal.Interface(), strings.ToLower(fieldName))
			otherBlocks = append(otherBlocks, blk)
		}
	}

	// top level resource block
	resourceBlock := hclwrite.NewBlock("resource", []string{typeName, resourceName})
	dst.AppendBlock(resourceBlock)

	// add metadata block as first child
	resourceBlock.Body().AppendBlock(metaBlock)

	// add all other blocks
	for _, blk := range otherBlocks {
		resourceBlock.Body().AppendBlock(blk)
	}
}

// EncodeAsBlock creates a new hclwrite.Block populated with the data from
// the given value, which must be a struct or pointer to struct.
func EncodeAsBlock(val interface{}, blockType string) *hclwrite.Block {
	rv := reflect.ValueOf(val)
	ty := rv.Type()
	if ty.Kind() == reflect.Ptr {
		rv = rv.Elem()
		ty = rv.Type()
	}
	if ty.Kind() != reflect.Struct && ty.Kind() != reflect.Map {
	}

	block := hclwrite.NewBlock(blockType, nil)
	switch ty.Kind() {
	case reflect.Struct:
		populateBody(rv, ty, block.Body())

	case reflect.Map:
		fmt.Println("encoding map %s", blockType)
		encodeMap(rv.Interface().(map[string]string), block.Body())
		// valTy, err := gocty.ImpliedType(rv.Interface())
		// if err != nil {
		// 	panic(fmt.Sprintf("cannot encode %T as HCL expression: %s", rv.Interface(), err))
		// }

		// val, err := gocty.ToCtyValue(rv.Interface(), valTy)
		// if err != nil {
		// 	// This should never happen, since we should always be able
		// 	// to decode into the implied type.
		// 	panic(fmt.Sprintf("failed to encode %T as %#v: %s", rv.Interface(), valTy, err))
		// }

		// block.Body().SetAttributeValue("map", val)

	default:
		panic(fmt.Sprintf("%s value is %s, not struct or map", blockType, ty.Kind()))
	}

	return block
}

func populateBody(rv reflect.Value, ty reflect.Type, dst *hclwrite.Body) {
	prevWasBlock := false

	for fieldIdx := 0; fieldIdx < rv.NumField(); fieldIdx++ {
		field := ty.Field(fieldIdx)
		fieldTy := field.Type
		fieldVal := rv.Field(fieldIdx)
		fieldName := field.Name

		if fieldTy.Kind() == reflect.Ptr {
			fieldTy = fieldTy.Elem()
			fieldVal = fieldVal.Elem()
		}

		if !fieldVal.CanSet() {
			fmt.Printf("%s: can't set\n", fieldName)
			continue // ignore unexported fields
		}

		fmt.Printf("SPEW %s: %s\n", fieldName, spew.Sdump(fieldVal))

		switch fieldTy.Kind() {
		case reflect.Struct:
			fmt.Printf("%s: struct -- %s\n", fieldName, spew.Sdump(fieldVal))
			if !fieldVal.IsValid() {
				continue // ignore (field value is nil pointer)
			}
			if fieldTy.Kind() == reflect.Ptr && fieldVal.IsNil() {
				continue // ignore
			}
			block := EncodeAsBlock(fieldVal.Interface(), fieldName)
			if !prevWasBlock {
				dst.AppendNewline()
				prevWasBlock = true
			}
			dst.AppendBlock(block)

		case reflect.Array:
			fallthrough
		case reflect.Slice:
			fmt.Printf("%s: array or slice\n", fieldName)
			s := fieldVal.Interface()
			typ := reflect.TypeOf(s).Elem()
			fmt.Printf("%sSlice Elem Type: %v\n", fieldName, typ)

		// case reflect.Map:

		default:
			fmt.Printf("%s: other\n", fieldName)
			// TODO: ignore empty values if omitempty tag is set on field

			if !fieldVal.IsValid() {
				continue // ignore (field value is nil pointer)
			}
			if fieldTy.Kind() == reflect.Ptr && fieldVal.IsNil() {
				continue // ignore
			}

			if prevWasBlock {
				dst.AppendNewline()
				prevWasBlock = false
			}

			valTy, err := gocty.ImpliedType(fieldVal.Interface())
			if err != nil {
				panic(fmt.Sprintf("cannot encode %T as HCL expression: %s", fieldVal.Interface(), err))
			}

			val, err := gocty.ToCtyValue(fieldVal.Interface(), valTy)
			if err != nil {
				// This should never happen, since we should always be able
				// to decode into the implied type.
				panic(fmt.Sprintf("failed to encode %T as %#v: %s", fieldVal.Interface(), valTy, err))
			}

			dst.SetAttributeValue(strings.ToLower(fieldName), val)
		}

		// if _, isAttr := tags.Attributes[name]; isAttr {

		// 	if exprType.AssignableTo(fieldTy) || attrType.AssignableTo(fieldTy) {
		// 		continue // ignore undecoded fields
		// 	}

		// } else { // must be a block, then
		// 	elemTy := fieldTy
		// 	isSeq := false
		// 	if elemTy.Kind() == reflect.Slice || elemTy.Kind() == reflect.Array {
		// 		isSeq = true
		// 		elemTy = elemTy.Elem()
		// 	}

		// 	if bodyType.AssignableTo(elemTy) || attrsType.AssignableTo(elemTy) {
		// 		continue // ignore undecoded fields
		// 	}
		// 	prevWasBlock = false

		// 	if isSeq {
		// 		l := fieldVal.Len()
		// 		for i := 0; i < l; i++ {
		// 			elemVal := fieldVal.Index(i)
		// 			if !elemVal.IsValid() {
		// 				continue // ignore (elem value is nil pointer)
		// 			}
		// 			if elemTy.Kind() == reflect.Ptr && elemVal.IsNil() {
		// 				continue // ignore
		// 			}
		// 			block := EncodeAsBlock(elemVal.Interface(), name)
		// 			if !prevWasBlock {
		// 				dst.AppendNewline()
		// 				prevWasBlock = true
		// 			}
		// 			dst.AppendBlock(block)
		// 		}
		// 	} else {

		// 	}
		// }
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
	}
}

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

func encodeMap2(m map[string]interface{}, dst *hclwrite.Body) {
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
