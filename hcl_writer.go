package main

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mitchellh/reflectwalk"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"k8s.io/apimachinery/pkg/runtime"
)

// WriteObject converts a Kubernetes runtime.Object to HCL
func WriteObject(obj runtime.Object, dst *hclwrite.Body) {
	w := NewWalker(dst)
	reflectwalk.Walk(obj, w)

	return
}

type walker struct {
	// debug logging helper
	indent string

	isTopLevel bool

	currentBlock     *blk
	currentField     reflect.StructField
	skipField        bool
	dst              *hclwrite.Body
	ignoreSliceElems bool
}

// blk is a wrapper for hclwrite.Block that allows to tag some extra
// info to each block.
type blk struct {
	parent *blk
	hcl    *hclwrite.Block

	// hasValue means a child field of this block had a non-nil / empty value
	// if this is false when CloseBlk() is called, the block won't be appended to parent
	hasValue bool
}

func NewWalker(dst *hclwrite.Body) *walker {
	w := &walker{}
	w.isTopLevel = true
	w.dst = dst

	return w
}

func (w *walker) StartNewBlk(hcl *hclwrite.Block) *blk {
	blk := &blk{
		parent: w.currentBlock,
		hcl:    hcl,
	}

	w.currentBlock = blk
	return blk
}

func (w *walker) CloseBlk() *blk {
	w.debug(fmt.Sprint("closing ", w.currentField.Name))
	parent := w.currentBlock.parent

	if parent == nil {
		w.dst.AppendBlock(w.currentBlock.hcl)

	} else {
		if w.currentBlock.hasValue {
			parent.hasValue = true
			parent.hcl.Body().AppendBlock(w.currentBlock.hcl)
			w.currentBlock.hcl.Body().AppendNewline()
		}

		w.currentBlock = parent
	}
	return w.currentBlock
}

// Enter is called by reflectwalk.Walk each time we enter a level
func (w *walker) Enter(l reflectwalk.Location) error {
	w.debug(fmt.Sprint("entering ", l))
	if l == reflectwalk.WalkLoc {
		w.isTopLevel = true
	}

	// increase indent
	if l == reflectwalk.Struct || l == reflectwalk.Slice || l == reflectwalk.Map {
		w.increaseIndent()
	}
	return nil
}

func (w *walker) Exit(l reflectwalk.Location) error {
	w.debug(fmt.Sprint("exiting ", l))
	switch l {
	case reflectwalk.Slice:
		w.ignoreSliceElems = false
		w.decreaseIndent()
		w.debug(fmt.Sprint("]"))

	case reflectwalk.Struct:
		fallthrough

	case reflectwalk.Map:
		w.CloseBlk()
		w.decreaseIndent()
		w.debug(fmt.Sprint("}"))

	}

	return nil
}

func (w *walker) Struct(v reflect.Value) error {
	if !v.CanInterface() {
		return nil
	}
	ty := reflect.TypeOf(v.Interface())
	w.debug(fmt.Sprintf("%s {\n", ty.Name()))

	if w.isTopLevel {
		w.isTopLevel = false
		typeName := ToTerraformResourceType(v)
		resName := ToTerraformResourceName(v)

		// create top level HCL block
		topLevelBlock := hclwrite.NewBlock("resource", []string{typeName, resName})
		w.StartNewBlk(topLevelBlock)

	} else {
		blockName := ToTerraformSubBlockName(w.currentField)
		hcl := hclwrite.NewBlock(blockName, nil)
		w.StartNewBlk(hcl)

	}

	return nil
}

func (w *walker) StructField(field reflect.StructField, v reflect.Value) error {
	if !v.IsValid() {
		w.debug(fmt.Sprint("skipping invalid ", field.Name))
		return reflectwalk.SkipEntry

	} else if ignoredField(field.Name) {
		w.debug(fmt.Sprint("ignoring ", field.Name))
		return reflectwalk.SkipEntry

	} else {
		w.currentField = field

	}
	return nil
}

func (w *walker) Primitive(v reflect.Value) error {
	if !w.ignoreSliceElems && v.CanAddr() && v.CanInterface() {
		w.debug(fmt.Sprintf("%s = %v (%T)[%s]", w.currentField.Name, v.Interface(), v.Interface(), w.currentField.Tag))

		if !IsZero(v) {
			w.currentBlock.hasValue = true
			w.currentBlock.hcl.Body().SetAttributeValue(
				ToTerraformAttributeName(w.currentField),
				convertCtyValue(v.Interface()),
			)
		}
	}
	return nil
}

func (w *walker) Map(m reflect.Value) error {
	w.debug(fmt.Sprintf("%s {\n", w.currentField.Name))

	blockName := ToTerraformSubBlockName(w.currentField)
	hcl := hclwrite.NewBlock(blockName, nil)
	w.StartNewBlk(hcl)

	return nil
}

func (w *walker) MapElem(m, k, v reflect.Value) error {
	w.debug(fmt.Sprintf("    %s = %v (%T)", k, v.Interface(), v.Interface()))

	if !IsZero(v) {
		w.currentBlock.hasValue = true
		w.currentBlock.hcl.Body().SetAttributeValue(
			k.String(),
			convertCtyValue(v.Interface()),
		)
	}

	return nil
}

func (w *walker) Slice(v reflect.Value) error {
	if !v.IsValid() {
		w.debug(fmt.Sprint("skipping invalid slice "))

	} else if IsZero(v) {
		w.debug(fmt.Sprint("skipping empty slice "))

	} else {
		w.debug(fmt.Sprintf("%s [\n", w.currentField.Name))
		// determine type of slice elements
		numEntries := v.Len()
		var vt reflect.Type
		if numEntries > 0 {
			w.currentBlock.hasValue = true
			vt = v.Index(0).Type()
		}

		switch {
		case vt.Kind() == reflect.Struct:
			// walk
		case vt.Kind() == reflect.Ptr:
			// walk

		default:
			valTy, err := gocty.ImpliedType(v.Interface())
			if err != nil {
				panic(fmt.Sprintf("cannot encode %T as HCL expression: %s", v.Interface(), err))
			}

			val, err := gocty.ToCtyValue(v.Interface(), valTy)
			if err != nil {
				// This should never happen, since we should always be able
				// to decode into the implied type.
				panic(fmt.Sprintf("failed to encode %T as %#v: %s", v.Interface(), valTy, err))
			}

			// primitive type
			w.currentBlock.hasValue = true
			w.currentBlock.hcl.Body().SetAttributeValue(
				ToTerraformAttributeName(w.currentField),
				val,
			)

			// don't need to walk through all Slice Elements, so return skip signal
			w.ignoreSliceElems = true
		}

	}

	return nil
}

func (w *walker) SliceElem(i int, v reflect.Value) error {
	// fmt.Printf("%s%v [\n", w.indent, v.Interface())
	return nil
}

func (w *walker) increaseIndent() {
	w.indent = w.indent + "  "
}

func (w *walker) decreaseIndent() {
	w.indent = w.indent[:len(w.indent)-2]
}

// func WriteObject(obj runtime.Object, dst *hclwrite.Body) {
// 	kubeKind := obj.GetObjectKind().GroupVersionKind().Kind
// 	kubeVersion := obj.GetObjectKind().GroupVersionKind().Version

// 	switch {
// 	case kubeKind == "ConfigMap":
// 		// WriteConfigMap(obj.(*corev1.ConfigMap), dst)

// 	case kubeKind == "Deployment" && kubeVersion == "v1":
// 		// WriteDeployment(obj.(*v1.Deployment), dst)

// 	default:
// 		// printFields(obj, "")
// 	}

// 	rv := reflect.ValueOf(obj)
// 	ty := rv.Type()
// 	if ty.Kind() == reflect.Ptr {
// 		rv = rv.Elem()
// 		ty = rv.Type()
// 	}

// 	var typeName string
// 	var resourceName string
// 	var metaBlock *hclwrite.Block
// 	var otherBlocks []*hclwrite.Block
// 	for i := 0; i < rv.NumField(); i++ {
// 		fieldVal := rv.Field(i)
// 		fieldName := ty.Field(i).Name
// 		switch fieldName {
// 		case "ObjectMeta":
// 			objMeta := fieldVal.Interface().(metav1.ObjectMeta)
// 			metaBlock = encodeMetadataBlock(&objMeta)
// 			resourceName = strcase.ToSnake(objMeta.Name)
// 		case "TypeMeta":
// 			typeMeta := fieldVal.Interface().(metav1.TypeMeta)
// 			typeName = "kubernetes_" + strcase.ToSnake(typeMeta.Kind)
// 		case "Status":
// 			continue
// 		// case "BinaryData":
// 		// 	// don't add to TF resource
// 		// 	continue
// 		default:
// 			// must some other field like 'spec' or 'data'
// 			blk := EncodeAsBlock(fieldVal.Interface(), strings.ToLower(fieldName))
// 			otherBlocks = append(otherBlocks, blk)
// 		}
// 	}

// 	// top level resource block
// 	resourceBlock := hclwrite.NewBlock("resource", []string{typeName, resourceName})
// 	dst.AppendBlock(resourceBlock)

// 	// add metadata block as first child
// 	resourceBlock.Body().AppendBlock(metaBlock)

// 	// add all other blocks
// 	for _, blk := range otherBlocks {
// 		resourceBlock.Body().AppendBlock(blk)
// 	}
// }

// // EncodeAsBlock creates a new hclwrite.Block populated with the data from
// // the given value, which must be a struct or pointer to struct.
// func EncodeAsBlock(val interface{}, blockType string) *hclwrite.Block {
// 	rv := reflect.ValueOf(val)
// 	ty := rv.Type()
// 	if ty.Kind() == reflect.Ptr {
// 		rv = rv.Elem()
// 		ty = rv.Type()
// 	}
// 	if ty.Kind() != reflect.Struct && ty.Kind() != reflect.Map {
// 	}

// 	block := hclwrite.NewBlock(blockType, nil)
// 	switch ty.Kind() {
// 	case reflect.Struct:
// 		populateBody(rv, ty, block.Body())

// 	case reflect.Map:
// 		fmt.Println("encoding map %s", blockType)
// 		encodeMap(rv.Interface().(map[string]string), block.Body())
// 		// valTy, err := gocty.ImpliedType(rv.Interface())
// 		// if err != nil {
// 		// 	panic(fmt.Sprintf("cannot encode %T as HCL expression: %s", rv.Interface(), err))
// 		// }

// 		// val, err := gocty.ToCtyValue(rv.Interface(), valTy)
// 		// if err != nil {
// 		// 	// This should never happen, since we should always be able
// 		// 	// to decode into the implied type.
// 		// 	panic(fmt.Sprintf("failed to encode %T as %#v: %s", rv.Interface(), valTy, err))
// 		// }

// 		// block.Body().SetAttributeValue("map", val)

// 	default:
// 		panic(fmt.Sprintf("%s value is %s, not struct or map", blockType, ty.Kind()))
// 	}

// 	return block
// }

// func populateBody(rv reflect.Value, ty reflect.Type, dst *hclwrite.Body) {
// 	prevWasBlock := false

// 	for fieldIdx := 0; fieldIdx < rv.NumField(); fieldIdx++ {
// 		field := ty.Field(fieldIdx)
// 		fieldTy := field.Type
// 		fieldVal := rv.Field(fieldIdx)
// 		fieldName := field.Name

// 		if fieldTy.Kind() == reflect.Ptr {
// 			fieldTy = fieldTy.Elem()
// 			fieldVal = fieldVal.Elem()
// 		}

// 		if !fieldVal.CanSet() {
// 			fmt.Printf("%s: can't set\n", fieldName)
// 			continue // ignore unexported fields
// 		}

// 		fmt.Printf("SPEW %s: %s\n", fieldName, spew.Sdump(fieldVal))

// 		switch fieldTy.Kind() {
// 		case reflect.Struct:
// 			fmt.Printf("%s: struct -- %s\n", fieldName, spew.Sdump(fieldVal))
// 			if !fieldVal.IsValid() {
// 				continue // ignore (field value is nil pointer)
// 			}
// 			if fieldTy.Kind() == reflect.Ptr && fieldVal.IsNil() {
// 				continue // ignore
// 			}
// 			block := EncodeAsBlock(fieldVal.Interface(), fieldName)
// 			if !prevWasBlock {
// 				dst.AppendNewline()
// 				prevWasBlock = true
// 			}
// 			dst.AppendBlock(block)

// 		case reflect.Array:
// 			fallthrough
// 		case reflect.Slice:
// 			fmt.Printf("%s: array or slice\n", fieldName)
// 			s := fieldVal.Interface()
// 			typ := reflect.TypeOf(s).Elem()
// 			fmt.Printf("%sSlice Elem Type: %v\n", fieldName, typ)

// 		// case reflect.Map:

// 		default:
// 			fmt.Printf("%s: other\n", fieldName)
// 			// TODO: ignore empty values if omitempty tag is set on field

// 			if !fieldVal.IsValid() {
// 				continue // ignore (field value is nil pointer)
// 			}
// 			if fieldTy.Kind() == reflect.Ptr && fieldVal.IsNil() {
// 				continue // ignore
// 			}

// 			if prevWasBlock {
// 				dst.AppendNewline()
// 				prevWasBlock = false
// 			}

// 			valTy, err := gocty.ImpliedType(fieldVal.Interface())
// 			if err != nil {
// 				panic(fmt.Sprintf("cannot encode %T as HCL expression: %s", fieldVal.Interface(), err))
// 			}

// 			val, err := gocty.ToCtyValue(fieldVal.Interface(), valTy)
// 			if err != nil {
// 				// This should never happen, since we should always be able
// 				// to decode into the implied type.
// 				panic(fmt.Sprintf("failed to encode %T as %#v: %s", fieldVal.Interface(), valTy, err))
// 			}

// 			dst.SetAttributeValue(strings.ToLower(fieldName), val)
// 		}

// 		// if _, isAttr := tags.Attributes[name]; isAttr {

// 		// 	if exprType.AssignableTo(fieldTy) || attrType.AssignableTo(fieldTy) {
// 		// 		continue // ignore undecoded fields
// 		// 	}

// 		// } else { // must be a block, then
// 		// 	elemTy := fieldTy
// 		// 	isSeq := false
// 		// 	if elemTy.Kind() == reflect.Slice || elemTy.Kind() == reflect.Array {
// 		// 		isSeq = true
// 		// 		elemTy = elemTy.Elem()
// 		// 	}

// 		// 	if bodyType.AssignableTo(elemTy) || attrsType.AssignableTo(elemTy) {
// 		// 		continue // ignore undecoded fields
// 		// 	}
// 		// 	prevWasBlock = false

// 		// 	if isSeq {
// 		// 		l := fieldVal.Len()
// 		// 		for i := 0; i < l; i++ {
// 		// 			elemVal := fieldVal.Index(i)
// 		// 			if !elemVal.IsValid() {
// 		// 				continue // ignore (elem value is nil pointer)
// 		// 			}
// 		// 			if elemTy.Kind() == reflect.Ptr && elemVal.IsNil() {
// 		// 				continue // ignore
// 		// 			}
// 		// 			block := EncodeAsBlock(elemVal.Interface(), name)
// 		// 			if !prevWasBlock {
// 		// 				dst.AppendNewline()
// 		// 				prevWasBlock = true
// 		// 			}
// 		// 			dst.AppendBlock(block)
// 		// 		}
// 		// 	} else {

// 		// 	}
// 		// }
// 	}
// }

// func printFields(obj interface{}, indent string) {
// 	fmt.Printf("%s---\n", indent)
// 	rv := reflect.ValueOf(obj)
// 	ty := rv.Type()
// 	if ty.Kind() == reflect.Ptr {
// 		rv = rv.Elem()
// 		ty = rv.Type()
// 	}

// 	fmt.Printf("%sType %s\n", indent, ty)
// 	fmt.Printf("%sKind %s\n", indent, ty.Kind())
// 	fmt.Printf("%sNumber of fields %d\n", indent, rv.NumField())
// 	indent = indent + "  "
// 	for i := 0; i < rv.NumField(); i++ {
// 		fmt.Printf("%s---\n", indent)
// 		fieldName := ty.Field(i).Name
// 		if ignoredField(fieldName) {
// 			continue
// 		}

// 		field := ty.Field(i)
// 		fieldTy := field.Type
// 		fieldVal := rv.Field(i)
// 		fmt.Printf("%sField Name: %s\n", indent, ty.Field(i).Name)
// 		fmt.Printf("%sField Type: %s\n", indent, fieldVal.Type())
// 		fmt.Printf("%sField Kind: %s\n", indent, fieldVal.Kind())
// 		fmt.Printf("%sField Tags: %s\n", indent, ty.Field(i).Tag)

// 		if fieldVal.Kind() == reflect.Struct {
// 			printFields(fieldVal.Interface(), indent+"  ")

// 		} else if fieldVal.Kind() == reflect.Slice {
// 			fmt.Printf("%sField Value: %v\n", indent, fieldVal)
// 			fmt.Printf("%sSLICE: %s -- %s\n", indent, spew.Sdump(fieldVal), spew.Sdump(fieldVal.Interface()))

// 			s := fieldVal.Interface()
// 			typ := reflect.TypeOf(s).Elem()
// 			fmt.Printf("%sSlice Elem Type: %v\n", indent, typ)

// 		} else {
// 			if fieldTy.Kind() == reflect.Ptr {
// 				fieldTy = fieldTy.Elem()
// 				fieldVal = fieldVal.Elem()
// 			}
// 			if !fieldVal.IsValid() {
// 				continue
// 			}
// 			fmt.Printf("%sField Value: %v\n", indent, fieldVal)

// 		}
// 	}
// }

// func encodeMetadataBlock(meta *metav1.ObjectMeta) *hclwrite.Block {
// 	blk := hclwrite.NewBlock("metadata", nil)
// 	blk.Body().SetAttributeValue("name", convertCtyValue(meta.Name))

// 	if meta.Namespace != "" {
// 		blk.Body().SetAttributeValue("namespace", convertCtyValue(meta.Namespace))
// 	}

// 	if len(meta.Labels) > 0 {
// 		lbls := hclwrite.NewBlock("labels", nil)
// 		encodeMap(meta.Labels, lbls.Body())
// 		blk.Body().AppendBlock(lbls)
// 	}

// 	if len(meta.Annotations) > 0 {
// 		anno := hclwrite.NewBlock("annotations", nil)
// 		encodeMap(meta.Annotations, anno.Body())
// 		blk.Body().AppendBlock(anno)
// 	}
// 	return blk
// }

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
		return cty.StringVal(fmt.Sprintf("%v", val))
	}
	return cty.NilVal
}

// func encodeMap(m map[string]string, dst *hclwrite.Body) {
// 	for k, v := range m {
// 		dst.SetAttributeValue(k, convertCtyValue(v))
// 	}
// }

// func encodeMap2(m map[string]interface{}, dst *hclwrite.Body) {
// 	for k, v := range m {
// 		dst.SetAttributeValue(k, convertCtyValue(v))
// 	}
// }

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

func (w *walker) debug(s string) {
	if debug {
		fmt.Printf("%s%s\n", w.indent, s)
	}
}
