package main

import (
	"github.com/sl1pm4t/k2tf/pkg/tfkschema"
	"strings"

	"github.com/hashicorp/hcl2/hclwrite"
	log "github.com/sirupsen/logrus"
	"github.com/zclconf/go-cty/cty"
)

// hclBlock is a wrapper for hclwrite.Block that allows tagging some extra
// metadata to each block.
type hclBlock struct {
	//
	name string

	//
	fieldName string

	// The parent hclBlock to this hclBlock
	parent *hclBlock

	// The wrapped HCL block
	hcl *hclwrite.Block

	// hasValue means a child field of this block had a non-nil / non-zero value.
	// If this is false when closeBlock() is called, the block won't be appended to
	// parent
	hasValue bool

	// inlined flags whether this block is "transparent" Some Structs in the
	// Kubernetes API structure are marked as "inline", meaning they don't create
	// a new block, and their child value is propagated up the hierarchy.
	// See v1.Volume as an example
	inlined bool

	// inlined flags whether this block is supported in the Terraform Provider schema
	// Unsupported blocks will be excluded from HCL rendering
	unsupported bool

	// isMap flags whether the output of this block will be map syntax rather than a sub-block
	// e.g.
	// mapName = {
	//   key = "value"
	// }
	// vs.
	// mapName {
	//   key = "value"
	// }
	// In TF0.12.0 Schema attributes of type schema.TypeMap must be written with the former syntax, and sub-blocks
	// most use the latter.
	// However, there are some cases where a Golang map on the Kubernetes object side is not defined
	// as schema.TypeMap on the Terraform side (e.g. Container.Limits) so isMap is used to track how this block
	// should be outputted.
	isMap  bool
	hclMap map[string]cty.Value
}

// A child block is adding a sub-block, write HCL to:
// - this hclBlock's hcl Body if this block is not inlined
// - parent's HCL body if this block is "inlined"
func (b *hclBlock) AppendBlock(hcl *hclwrite.Block) {
	if b.inlined {
		// append to parent
		b.parent.AppendBlock(hcl)

	} else {
		b.hcl.Body().AppendBlock(hcl)

	}
}

// A child block is adding an attribute, write HCL to:
// - this hclBlock's hcl Body if this block is not inlined
// - parent's HCL body if this block is "inlined"
func (b *hclBlock) SetAttributeValue(name string, val cty.Value) {
	if b.isMap {
		if b.hclMap == nil {
			b.hclMap = map[string]cty.Value{name: val}
		} else {
			b.hclMap[name] = val
		}

	} else if includeUnsupported || tfkschema.IsAttributeSupported(b.FullSchemaName()+"."+name) {
		if b.inlined {
			// append to parent
			b.parent.SetAttributeValue(name, val)
		} else {
			b.hcl.Body().SetAttributeValue(name, val)
		}
	} else {
		log.Debugf("skipping attribute: %s - not supported by provider", name)

	}
}

func (b *hclBlock) FullSchemaName() string {
	parentName := ""
	if b.parent != nil {
		parentName = b.parent.FullSchemaName()
	}

	if b.inlined {
		return parentName
	}
	return strings.TrimLeft(parentName+"."+b.name, ".")
}

func (b *hclBlock) isSupportedAttribute() bool {
	return tfkschema.IsAttributeSupported(b.FullSchemaName())
}

func (b *hclBlock) isRequired() bool {
	if b.parent == nil {
		// top level resource block is always required.
		return true
	}

	required := tfkschema.IsAttributeRequired(b.FullSchemaName())

	if required && !b.hasValue && !b.parent.isRequired() {
		// If current attribute has no value, only flag as required if parent(s) are also required.
		// This is to match how Terraform handles the Required flag of nested attributes.
		required = false
	}

	return required
}

func (b *hclBlock) FullFieldName() string {
	parentName := ""
	if b.parent != nil {
		parentName = b.parent.FullFieldName()
	}

	if b.inlined {
		return parentName
	}
	return strings.TrimLeft(parentName+"."+b.fieldName, ".")
}
