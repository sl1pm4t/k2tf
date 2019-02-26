package main

import (
	"fmt"
	"reflect"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mitchellh/reflectwalk"
	"github.com/rs/zerolog/log"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

// WriteObject converts a Kubernetes runtime.Object to HCL
func WriteObject(obj runtime.Object, dst *hclwrite.Body) {
	w := NewObjectWalker(obj, dst)
	reflectwalk.Walk(obj, w)

	return
}

// ObjectWalker implements reflectwalk.Walker interface
// It used to "walk" the Kubernetes API Objects structure and generate
// an HCL document based on the values defined.
type ObjectWalker struct {
	// The Kubernetes API Object to be walked
	RuntimeObject runtime.Object

	// Terraform resource type (e.g. kubernetes_pod)
	ResourceType string

	// debug logging helper
	depth  int
	indent string

	// top level HCL
	isTopLevel bool
	dst        *hclwrite.Body

	// sub block tracking
	currentBlock *hclBlock
	fields       []*reflect.StructField
	currentField *reflect.StructField

	// slices of structs
	sliceField       *reflect.StructField
	sliceElemTypes   []reflect.Type
	ignoreSliceElems bool
}

// NewObjectWalker returns a new ObjectWalker object
// dst is the hclwrite.Body where HCL blocks will be appended.
func NewObjectWalker(obj runtime.Object, dst *hclwrite.Body) *ObjectWalker {
	w := &ObjectWalker{
		RuntimeObject: obj,
		isTopLevel:    true,
		dst:           dst,
	}

	return w
}

func (w *ObjectWalker) setCurrentField(f *reflect.StructField) {
	if f != nil {
		w.debugf("setting currentField to %s", f.Name)
		w.currentField = f
	}
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

type NilSliceElemType struct{}

func (w *ObjectWalker) sliceElemType() reflect.Type {
	var result reflect.Type
	if len(w.sliceElemTypes) > 0 {
		result = w.sliceElemTypes[len(w.sliceElemTypes)-1]
	} else {
		result = reflect.TypeOf(NilSliceElemType{})
	}

	w.debugf("sliceElemType %s", result.Name())
	return result
}

// openBlk opens a new HCL resource block or sub-block
// It creates a hclBlock object so we can track hierarchy of blocks
// within the resource tree
func (w *ObjectWalker) openBlk(name string, hcl *hclwrite.Block) *hclBlock {
	w.debugf("opening hclBlock for field: %s", name)
	blk := &hclBlock{
		name:   name,
		parent: w.currentBlock,
		hcl:    hcl,
	}

	w.currentBlock = blk
	return blk
}

// closeBlk writes the generated HCL to the hclwriter
func (w *ObjectWalker) closeBlk() *hclBlock {
	w.debugf("closing hclBlock: %s | [%v]", w.currentBlock.name, w.currentBlock)

	parent := w.currentBlock.parent
	current := w.currentBlock

	// TODO: move append logic to hcl_block to be consistent
	if parent == nil {
		// we are closing the top level block, write directly to HCL File
		w.dst.AppendBlock(current.hcl)

	} else {
		if !includeUnsupported && current.unsupported {
			// don't append this block or child blocks
			log.Warn().Str("attribute", current.FullSchemaName()).Msg("excluding attribute because it's unsupported in Terraform schema")

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
	w.depth++
	w.debug(fmt.Sprint("entering ", l))

	switch l {
	case reflectwalk.Slice:
		w.increaseIndent()

	case reflectwalk.Struct:
		fallthrough
	case reflectwalk.Map:
		w.increaseIndent()

	}

	return nil
}

// Exit is called by reflectwalk each time it exits from a reflectwalk.Location
func (w *ObjectWalker) Exit(l reflectwalk.Location) error {
	w.depth--
	switch l {
	case reflectwalk.Slice:
		if !w.ignoreSliceElems {
			w.sliceElemTypePop()
		}
		w.ignoreSliceElems = false
		w.decreaseIndent()

	case reflectwalk.Struct:
		fallthrough

	case reflectwalk.Map:
		w.closeBlk()
		w.decreaseIndent()

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
		return nil
	}

	ty := reflect.TypeOf(v.Interface())

	if w.isTopLevel {
		// we need to create the top level HCL block
		// e.g. resource "kubernetes_pod" "name" {
		w.isTopLevel = false
		w.ResourceType = ToTerraformResourceType(v)
		resName := ToTerraformResourceName(w.RuntimeObject)

		// create the HCL block
		topLevelBlock := hclwrite.NewBlock("resource", []string{w.ResourceType, resName})
		w.openBlk(w.ResourceType, topLevelBlock)

	} else {
		// this struct is a sub-block, create a new HCL block and add to parent
		field := w.currentField

		if w.sliceElemType() == ty {
			// when iterating over a slice of complex types, the block name is based on the
			// Slices StructField data instead of the Slice element.
			w.debug("using sliceField instead of currentField")
			field = w.sliceField
		}

		blockName := ToTerraformSubBlockName(field)
		w.debugf("creating blk [%s] for field [%s]", blockName, field.Name)
		blk := w.openBlk(blockName, hclwrite.NewBlock(blockName, nil))

		// Skip some Kubernetes complex types that should be treated as Primitives.
		// Do this after opening the Block above because reflectwalk will
		// still call Exit for this struct and we need the calls to closeBlk() to marry up
		switch v.Interface().(type) {
		case resource.Quantity:
			return reflectwalk.SkipEntry
		}

		blk.inlined = IsInlineStruct(field)

		var err error
		supported, err := SchemaSupportsAttribute(blk.FullSchemaName())
		if err != nil {
			log.Warn().Str("error", err.Error()).Msg("error while validating attribute against schema")
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
		w.debug(fmt.Sprintf("%s = %v (%T)[%s]", w.currentField.Name, v.Interface(), v.Interface(), w.currentField.Tag))

		if !IsZero(v) {
			w.currentBlock.hasValue = true
			w.currentBlock.SetAttributeValue(
				ToTerraformAttributeName(w.currentField),
				w.convertCtyValue(v.Interface()),
			)
		}
	}
	return nil
}

// Map is called everytime reflectwalk enters a Map
func (w *ObjectWalker) Map(m reflect.Value) error {
	blockName := ToTerraformSubBlockName(w.currentField)
	hcl := hclwrite.NewBlock(blockName, nil)
	w.openBlk(blockName, hcl)

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

		w.sliceElemTypePush(vt)

		switch {
		case vt.Kind() == reflect.Struct:
			fallthrough
		case vt.Kind() == reflect.Ptr:
			w.debug("slice of Pointers / Structs")
			w.sliceField = w.currentField
			// walk

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
				ToTerraformAttributeName(w.currentField),
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
	w.debugf("Elem: %v", v.Interface())
	return nil
}

func (w *ObjectWalker) increaseIndent() {
	w.indent = w.indent + "  "
}

func (w *ObjectWalker) decreaseIndent() {
	w.indent = w.indent[:len(w.indent)-2]
}

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

	default:
		if s, ok := val.(fmt.Stringer); ok {
			return cty.StringVal(s.String())
		}

		log.Warn().Msg(fmt.Sprintf("unhandled variable type: %T", val))

		// last resort
		return cty.StringVal(fmt.Sprintf("%s", val))
	}
	return cty.NilVal
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

func (w *ObjectWalker) info(s string) {
	log.Info().
		Str("rtype", w.ResourceType).
		Str("current", w.currentBlock.FullSchemaName()).
		Msg(s)
}

func (w *ObjectWalker) infof(format string, a ...interface{}) {
	w.info(fmt.Sprintf(format, a...))
}

func (w *ObjectWalker) debug(s string) {
	e := log.Debug().
		Str("rtype", w.ResourceType)

	if w.currentBlock != nil {
		e = e.Str("current", w.currentBlock.FullSchemaName())
	}

	e.Msg(s)
}

func (w *ObjectWalker) debugf(format string, a ...interface{}) {
	w.debug(fmt.Sprintf(format, a...))
}
