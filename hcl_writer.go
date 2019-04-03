package main

import (
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/rs/zerolog"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mitchellh/reflectwalk"
	"github.com/rs/zerolog/log"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

// WriteObject converts a Kubernetes runtime.Object to HCL
func WriteObject(obj runtime.Object, dst *hclwrite.Body) error {
	w, err := NewObjectWalker(obj, dst)
	if err != nil {
		return err
	}
	reflectwalk.Walk(obj, w)

	return nil
}

// ObjectWalker implements reflectwalk.Walker interfaces
// It's used to "walk" the Kubernetes API Objects structure and generate
// an HCL document based on the values defined.
type ObjectWalker struct {
	// The Kubernetes API Object to be walked
	RuntimeObject runtime.Object

	// Terraform resource type (e.g. kubernetes_pod)
	resourceType string
	// Terraform resource name (adapted from ObjectMeta name attribute)
	resourceName string

	// debug logging helpers
	indent string

	// top level HCL
	isTopLevel bool
	dst        *hclwrite.Body

	// sub block tracking
	currentBlock *hclBlock
	fields       []*reflect.StructField
	currentField *reflect.StructField

	// slices of structs
	slices           []*reflect.StructField
	sliceField       *reflect.StructField
	sliceElemTypes   []reflect.Type
	ignoreSliceElems bool
}

// NewObjectWalker returns a new ObjectWalker object
// dst is the hclwrite.Body where HCL blocks will be appended.
func NewObjectWalker(obj runtime.Object, dst *hclwrite.Body) (*ObjectWalker, error) {
	if obj == nil {
		return nil, fmt.Errorf("obj cannot be nil")
	}

	w := &ObjectWalker{
		RuntimeObject: obj,
		isTopLevel:    true,
		dst:           dst,
	}

	return w, nil
}

func (w *ObjectWalker) setCurrentField(f *reflect.StructField) {
	if f != nil {
		w.debugf("setting currentField to %s", f.Name)
		w.currentField = f
	}
}

func (w *ObjectWalker) setCurrentSlice(f *reflect.StructField) {
	if f != nil {
		w.debugf("setting setCurrentSlice to %s", f.Name)
		w.sliceField = f
	}
}

func (w *ObjectWalker) currentSlice() *reflect.StructField {
	if len(w.slices) > 0 {
		return w.slices[len(w.slices)-1]
	}

	return nil
}

func (w *ObjectWalker) fieldPop() *reflect.StructField {
	result := w.fields[len(w.fields)-1]
	w.fields = w.fields[:len(w.fields)-1]

	w.debugf("fieldPop %s", result.Name)
	w.setCurrentField(result)
	return result
}

func (w *ObjectWalker) fieldPush(v *reflect.StructField) {
	w.fields = append(w.fields, v)
	w.debugf("fieldPush %s", v.Name)
	w.setCurrentField(v)
}

func (w *ObjectWalker) slicePop() *reflect.StructField {
	result := w.slices[len(w.slices)-1]
	w.slices = w.slices[:len(w.slices)-1]

	w.debugf("slicePop %s", result.Name)
	w.setCurrentSlice(result)
	return result
}

func (w *ObjectWalker) slicePush(v *reflect.StructField) {
	w.slices = append(w.slices, v)
	w.debugf("slicePush %s", v.Name)
	w.setCurrentSlice(v)
}

func (w *ObjectWalker) sliceType() reflect.Type {
	var result reflect.Type
	currSlice := w.currentSlice()
	if currSlice != nil {
		result = currSlice.Type
		w.debugf("sliceType %s", result.Name())
	} else {
		result = reflect.TypeOf(nil)
		w.debug("sliceType nil")
	}

	return result
}

func (w *ObjectWalker) sliceElemTypePush(ty reflect.Type) {
	w.sliceElemTypes = append(w.sliceElemTypes, ty)
	w.debugf("sliceElemTypePush %s", ty.Name())
}

func (w *ObjectWalker) sliceElemTypePop() reflect.Type {
	result := w.sliceElemTypes[len(w.sliceElemTypes)-1]
	w.sliceElemTypes = w.sliceElemTypes[:len(w.sliceElemTypes)-1]

	w.debugf("sliceElemTypePop %s", result.Name())
	return result
}

func (w *ObjectWalker) sliceElemType() reflect.Type {
	var result reflect.Type
	if len(w.sliceElemTypes) > 0 {
		result = w.sliceElemTypes[len(w.sliceElemTypes)-1]
	} else {
		result = reflect.TypeOf(struct{}{})
	}

	w.debugf("sliceElemType %s", result.Name())
	return result
}

// openBlk opens a new HCL resource block or sub-block
// It creates a hclBlock object so we can track hierarchy of blocks
// within the resource tree
func (w *ObjectWalker) openBlk(name, fieldName string, hcl *hclwrite.Block) *hclBlock {
	w.debugf("opening hclBlock for field: %s", name)
	blk := &hclBlock{
		name:      name,
		fieldName: fieldName,
		parent:    w.currentBlock,
		hcl:       hcl,
	}

	w.currentBlock = blk
	return blk
}

// closeBlk writes the generated HCL to the hclwriter
func (w *ObjectWalker) closeBlk() *hclBlock {
	w.debugf("closing hclBlock: %s", w.currentBlock.name)

	parent := w.currentBlock.parent
	current := w.currentBlock

	// TODO: move append logic to hcl_block to be consistent
	if parent == nil {
		// we are closing the top level block, write directly to HCL File
		w.dst.AppendBlock(current.hcl)

	} else {
		if !includeUnsupported && current.unsupported {
			// don't append this block or child blocks
			w.warn().
				Str("attr", current.FullSchemaName()).
				Str("field", current.FullFieldName()).
				Msg("excluding attribute - not found in Terraform schema")

		} else if current.hasValue {
			// communicate back up the tree that we found a non-zero value
			parent.hasValue = true

			if !current.inlined {
				parent.AppendBlock(current.hcl)
			}
		}

		w.currentBlock = parent
	}
	return w.currentBlock
}

// Enter is called by reflectwalk.Walk each time we enter a level
func (w *ObjectWalker) Enter(l reflectwalk.Location) error {
	w.debug(fmt.Sprint("entering ", l))

	return nil
}

// Exit is called by reflectwalk each time it exits from a reflectwalk.Location
func (w *ObjectWalker) Exit(l reflectwalk.Location) error {
	switch l {
	case reflectwalk.Slice:
		if !w.ignoreSliceElems {
			w.sliceElemTypePop()
		}
		w.ignoreSliceElems = false
		w.slicePop()

	case reflectwalk.Struct:
		fallthrough

	case reflectwalk.Map:
		w.closeBlk()

	case reflectwalk.StructField:
		w.fieldPop()
	}

	w.debugf("exiting %s", l)
	return nil
}

// Struct is called every time reflectwalk enters a Struct
//
func (w *ObjectWalker) Struct(v reflect.Value) error {
	if !v.CanInterface() {
		w.debugf("skipping Struct [field: %s, type: %s] - CanInterface() = false", w.currentField.Name, v.Type())
		return nil
	}

	ty := reflect.TypeOf(v.Interface())

	if w.isTopLevel {
		// we need to create the top level HCL block
		// e.g. resource "kubernetes_pod" "name" {
		topLevelBlock := hclwrite.NewBlock("resource", []string{w.ResourceType(), w.ResourceName()})
		w.openBlk(w.ResourceType(), typeMeta(w.RuntimeObject).Kind, topLevelBlock)
		w.isTopLevel = false

	} else {
		// this struct is a sub-block, create a new HCL block and add to parent
		field := w.currentField

		if w.sliceElemType() == ty || w.sliceType() == ty {
			// when iterating over a slice of complex types, the block name is based on the
			// Slices StructField data instead of the Slice element.
			w.debug("using sliceField instead of currentField")
			field = w.currentSlice()
		}

		blockName := ToTerraformSubBlockName(field, w.currentBlock.FullSchemaName())
		w.debugf("creating blk [%s] for field [%s]", blockName, field.Name)
		blk := w.openBlk(blockName, field.Name, hclwrite.NewBlock(blockName, nil))

		// Skip some Kubernetes complex types that should be treated as Primitives.
		// Do this after opening the Block above because reflectwalk will
		// still call Exit for this struct and we need the calls to closeBlk() to marry up
		// TODO: figure out a uniform way to handle these cases
		switch v.Interface().(type) {
		case resource.Quantity:
			return reflectwalk.SkipEntry
		case intstr.IntOrString:
			ios := v.Interface().(intstr.IntOrString)
			if ios.IntVal > 0 || ios.StrVal != "" {
				blk.hasValue = false
				blk.parent.SetAttributeValue(blockName, w.convertCtyValue(v.Interface()))
				blk.parent.hasValue = true
			}
			return reflectwalk.SkipEntry
		}

		blk.inlined = IsInlineStruct(field)

		var err error
		supported, err := IsAttributeSupported(blk.FullSchemaName())
		if err != nil && err != errAttrNotFound {
			w.warn().Str("error", err.Error()).Msg("error while validating attribute against schema")
		}
		blk.unsupported = !supported
	}

	return nil
}

// StructField is called by reflectwalk whenever it enters a field of a struct.
// We ignore Invalid fields, or some API fields we don't need to convert to HCL.
// The rest are added to the StuctField stack so we have access to the
// StructField data in other funcs.
func (w *ObjectWalker) StructField(field reflect.StructField, v reflect.Value) error {
	if !v.IsValid() {
		w.debug(fmt.Sprint("skipping invalid ", field.Name))
		return reflectwalk.SkipEntry

	} else if ignoredField(field.Name) {
		w.debug(fmt.Sprint("ignoring ", field.Name))
		return reflectwalk.SkipEntry

	} else {
		w.fieldPush(&field)

	}
	return nil
}

// Primitive is called whenever reflectwalk visits a Primitive value.
// If it's not a zero value, add an Attribute to the current HCL Block.
func (w *ObjectWalker) Primitive(v reflect.Value) error {
	if !w.ignoreSliceElems && v.CanAddr() && v.CanInterface() {
		w.debug(fmt.Sprintf("%s = %v (%T)", w.currentField.Name, v.Interface(), v.Interface()))

		if !IsZero(v) {
			w.currentBlock.hasValue = true
			w.currentBlock.SetAttributeValue(
				ToTerraformAttributeName(w.currentField, w.currentBlock.FullSchemaName()),
				w.convertCtyValue(v.Interface()),
			)
		}
	}
	return nil
}

// Map is called everytime reflectwalk enters a Map
func (w *ObjectWalker) Map(m reflect.Value) error {
	blockName := ToTerraformSubBlockName(w.currentField, w.currentBlock.FullSchemaName())
	hcl := hclwrite.NewBlock(blockName, nil)
	w.openBlk(blockName, w.currentField.Name, hcl)

	return nil
}

// MapElem is called everytime reflectwalk enters a Map element
func (w *ObjectWalker) MapElem(m, k, v reflect.Value) error {
	w.debug(fmt.Sprintf("    %s = %v (%T)", k, v.Interface(), v.Interface()))

	if !IsZero(v) {
		w.currentBlock.hasValue = true
		w.currentBlock.hcl.Body().SetAttributeValue(
			NormalizeTerraformMapKey(k.String()),
			w.convertCtyValue(v.Interface()),
		)
	}

	return nil
}

// Slice implements reflectwalk.SliceWalker interface
func (w *ObjectWalker) Slice(v reflect.Value) error {
	w.slicePush(w.currentField)
	if !v.IsValid() {
		w.debug(fmt.Sprint("skipping invalid slice "))
		w.ignoreSliceElems = true

	} else if IsZero(v) {
		w.debug(fmt.Sprint("skipping empty slice "))
		w.ignoreSliceElems = true

	} else {
		// determine type of slice elements
		numEntries := v.Len()
		var vt reflect.Type
		if numEntries > 0 {
			w.currentBlock.hasValue = true
			vt = v.Index(0).Type()
		}

		switch {
		case vt.Kind() == reflect.Struct:
			fallthrough
		case vt.Kind() == reflect.Ptr:
			w.debugf("slice of Pointers / Structs")
			w.sliceElemTypePush(vt)
			// walk elements

		default:
			valTy, err := gocty.ImpliedType(v.Interface())
			if err != nil {
				log.Panic().Interface("cannot encode %T as HCL expression", v.Interface()).Err(err)
			}

			val, err := gocty.ToCtyValue(v.Interface(), valTy)
			if err != nil {
				// This should never happen, since we should always be able
				// to decode into the implied type.
				log.Panic().Interface("failed to encode", v.Interface()).Interface("as %#v", valTy).Err(err)
			}

			// primitive type
			w.currentBlock.hasValue = true
			w.currentBlock.hcl.Body().SetAttributeValue(
				ToTerraformAttributeName(w.currentField, w.currentBlock.FullSchemaName()),
				val,
			)

			// don't need to walk through all Slice Elements
			w.ignoreSliceElems = true
		}

	}

	return nil
}

// SliceElem implements reflectwalk.SliceWalker interface
func (w *ObjectWalker) SliceElem(i int, v reflect.Value) error {
	w.debugf("Elem %d: %T", i, v.Interface())
	return nil
}

// convertCtyValue takes an interface and converts to HCL types
func (w *ObjectWalker) convertCtyValue(val interface{}) cty.Value {
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
			ctyMap[k] = w.convertCtyValue(v)
		}

		return cty.ObjectVal(ctyMap)
	case resource.Quantity:
		qty := val.(resource.Quantity)
		qtyPtr := &qty
		return cty.StringVal(qtyPtr.String())

	case intstr.IntOrString:
		ios := val.(intstr.IntOrString)
		iosPtr := &ios
		return cty.StringVal(iosPtr.String())

	default:
		if s, ok := val.(fmt.Stringer); ok {
			return cty.StringVal(s.String())
		}

		log.Debug().Msg(fmt.Sprintf("unhandled variable type: %T", val))

		// last resort
		return cty.StringVal(fmt.Sprintf("%s", val))
	}
}

var ignoredFields = []string{
	"CreationTimestamp",
	"DeletionTimestamp",
	"Generation",
	"OwnerReferences",
	"ResourceVersion",
	"SelfLink",
	"TypeMeta",
	"Status",
	"UID",
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

func (w *ObjectWalker) log(s string, e *zerolog.Event) {
	e.
		Str("type", w.ResourceType()).
		Str("name", w.ResourceName())

	if w.currentField != nil {
		e.Str("field", w.currentField.Name)
	}
	if w.currentSlice() != nil {
		e.Str("slice", w.currentSlice().Name)
	}
	e.Msg(s)
}

func (w *ObjectWalker) info(s string) {
	w.log(s, log.Info())
}

func (w *ObjectWalker) infof(format string, a ...interface{}) {
	w.info(fmt.Sprintf(format, a...))
}

func (w *ObjectWalker) debug(s string) {
	w.log(s, log.Debug())
}

func (w *ObjectWalker) debugf(format string, a ...interface{}) {
	w.debug(fmt.Sprintf(format, a...))
}

func (w *ObjectWalker) warn() *zerolog.Event {
	return w.decorateEvent(log.Warn())
}

func (w *ObjectWalker) decorateEvent(e *zerolog.Event) *zerolog.Event {
	e.
		Str("name", w.ResourceName()).
		Str("type", w.ResourceType())

	return e
}

// ResourceName returns the Terraform Resource name for the Kubernetes Object
func (w *ObjectWalker) ResourceName() string {
	if w.resourceName == "" {
		w.resourceName = ToTerraformResourceName(w.RuntimeObject)
	}

	return w.resourceName
}

// ResourceType returns the Terraform Resource type for the Kubernetes Object
func (w *ObjectWalker) ResourceType() string {
	if w.resourceType == "" {
		w.resourceType = ToTerraformResourceType(w.RuntimeObject)
	}

	return w.resourceType
}
