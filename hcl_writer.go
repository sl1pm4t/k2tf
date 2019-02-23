package main

import (
	"fmt"
	"os"
	"reflect"

	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/mitchellh/reflectwalk"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
)

// WriteObject converts a Kubernetes runtime.Object to HCL
func WriteObject(obj runtime.Object, dst *hclwrite.Body) {
	w := NewObjectWalker(dst)
	reflectwalk.Walk(obj, w)

	return
}

// ObjectWalker implements reflectwalk.Walker interface
// It used to "walk" the Kubernetes API Objects structure and generate
// an HCL document based on the values defined.
type ObjectWalker struct {
	// debug logging helper
	indent string

	isTopLevel bool

	currentBlock     *blk
	currentField     reflect.StructField
	dst              *hclwrite.Body
	ignoreSliceElems bool
}

// blk is a wrapper for hclwrite.Block that allows to tag some extra
// info to each block.
type blk struct {
	parent *blk
	hcl    *hclwrite.Block

	// hasValue means a child field of this block had a non-nil / empty value
	// if this is false when closeBlk() is called, the block won't be appended to parent
	hasValue bool

	// inlined flags whether this block is "transparent"
	// Some Structs in the Kubernetes API structure are marked as "inline",
	// meaning they don't create a new block, and their child value is propagated up the hierarchy.
	// See v1.Volume as an example
	inlined bool
}

// NewObjectWalker returns a new ObjectWalker object
// dst is the hclwrite.Body where HCL blocks will be appended.
func NewObjectWalker(dst *hclwrite.Body) *ObjectWalker {
	w := &ObjectWalker{}
	w.isTopLevel = true
	w.dst = dst

	return w
}

// openBlk opens a new HCL resource block or sub-block
// It creates a blk object so we can track hierarchy of blocks
// within the resource tree
func (w *ObjectWalker) openBlk(hcl *hclwrite.Block) *blk {
	w.debugf("opening blk for field %s", w.currentField.Name)
	blk := &blk{
		parent: w.currentBlock,
		hcl:    hcl,
	}

	w.currentBlock = blk
	return blk
}

// closeBlk writes the generated HCL to the hclwriter
func (w *ObjectWalker) closeBlk() *blk {
	w.debugf("closing %s", w.currentField.Name)

	parent := w.currentBlock.parent
	current := w.currentBlock

	if parent == nil {
		// we are closing the top level block, write directly to HCL File
		w.dst.AppendBlock(current.hcl)

	} else {
		// we are closing a sub-block, write HCL to either:
		// - parent Blocks HCL body in most cases
		// - parent's parents HCL body if our parent is "inlined"
		// - do nothing if the current blk is inlined
		if current.hasValue {
			// communicate back up the tree that we found a non-zero value
			parent.hasValue = true

			// Append HCL block in the appropriate location
			if parent.inlined {
				w.debugf("appending HCL block to grandparent")
				parent.parent.hcl.Body().AppendBlock(current.hcl)
				current.hcl.Body().AppendNewline()

			} else if current.inlined {
				w.debugf("closing inlined blk")
				// do nothing

			} else {
				w.debugf("appending HCL block to parent")
				parent.hcl.Body().AppendBlock(current.hcl)
				current.hcl.Body().AppendNewline()

			}
		}

		w.currentBlock = parent
	}
	return w.currentBlock
}

// Enter is called by reflectwalk.Walk each time we enter a level
func (w *ObjectWalker) Enter(l reflectwalk.Location) error {
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

// Exit is called by reflectwalk each time it exits from a
func (w *ObjectWalker) Exit(l reflectwalk.Location) error {
	w.debug(fmt.Sprint("exiting ", l))
	switch l {
	case reflectwalk.Slice:
		w.ignoreSliceElems = false
		w.decreaseIndent()
		w.debug(fmt.Sprint("]"))

	case reflectwalk.Struct:
		fallthrough

	case reflectwalk.Map:
		w.closeBlk()
		w.decreaseIndent()
		w.debug(fmt.Sprint("}"))

	}

	return nil
}

func (w *ObjectWalker) Struct(v reflect.Value) error {
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
		w.openBlk(topLevelBlock)

	} else {
		blockName := ToTerraformSubBlockName(w.currentField)
		w.debugf("creating blk [%s] for field [%s]", blockName, w.currentField.Name)
		hcl := hclwrite.NewBlock(blockName, nil)
		blk := w.openBlk(hcl)
		blk.inlined = IsInlineStruct(w.currentField)

	}

	// skip some Kubernetes structs that should be treated as Primitives instead
	// we do this after opening the Block above because reflectwalk will still call
	// Exit for this struct and we need the block Closes to marry up.
	switch v.Interface().(type) {
	case resource.Quantity:
		return reflectwalk.SkipEntry
		// case resource.Quantity:
		// 	return reflectwalk.SkipEntry
	}

	return nil
}

func (w *ObjectWalker) StructField(field reflect.StructField, v reflect.Value) error {
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

func (w *ObjectWalker) Primitive(v reflect.Value) error {
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

// Map is called everytime reflect enters a Map StructField
func (w *ObjectWalker) Map(m reflect.Value) error {
	w.debug(fmt.Sprintf("%s {\n", w.currentField.Name))

	blockName := ToTerraformSubBlockName(w.currentField)
	hcl := hclwrite.NewBlock(blockName, nil)
	w.openBlk(hcl)

	return nil
}

func (w *ObjectWalker) MapElem(m, k, v reflect.Value) error {
	w.debug(fmt.Sprintf("    %s = %v (%T)", k, v.Interface(), v.Interface()))

	if !IsZero(v) {
		w.currentBlock.hasValue = true
		w.currentBlock.hcl.Body().SetAttributeValue(
			NormalizeTerraformMapKey(k.String()),
			convertCtyValue(v.Interface()),
		)
	}

	return nil
}

func (w *ObjectWalker) Slice(v reflect.Value) error {
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

func (w *ObjectWalker) SliceElem(i int, v reflect.Value) error {
	// fmt.Printf("%s%v [\n", w.indent, v.Interface())
	return nil
}

func (w *ObjectWalker) increaseIndent() {
	w.indent = w.indent + "  "
}

func (w *ObjectWalker) decreaseIndent() {
	w.indent = w.indent[:len(w.indent)-2]
}

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
	case resource.Quantity:
		qty := val.(resource.Quantity)
		qtyPtr := &qty
		return cty.StringVal(qtyPtr.String())

	default:
		fmt.Fprintf(os.Stderr, "[!] WARN: unhandled variable type: %T \n", val)

		if s, ok := val.(fmt.Stringer); ok {
			return cty.StringVal(s.String())
		}
		// return cty.StringVal(fmt.Sprintf("%v", val))
	}
	return cty.NilVal
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

func (w *ObjectWalker) debug(s string) {
	if debug {
		fmt.Fprintf(os.Stderr, "%s%s\n", w.indent, s)
	}
}

func (w *ObjectWalker) debugf(format string, a ...interface{}) {
	w.debug(fmt.Sprintf(format, a...))
}
