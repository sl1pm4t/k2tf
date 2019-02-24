package main

import (
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// hclBlock is a wrapper for hclwrite.Block that allows tagging some extra
// metadata to each block.
type hclBlock struct {
	name string

	// The parent hclBlock to this hclBlock
	parent *hclBlock

	// The wrapped HCL block
	hcl *hclwrite.Block

	// hasValue means a child field of this block had a non-nil / non-zero value.
	// If this is false when closeBlk() is called, the block won't be appended to
	// parent
	hasValue bool

	// inlined flags whether this block is "transparent" Some Structs in the
	// Kubernetes API structure are marked as "inline", meaning they don't create
	// a new block, and their child value is propagated up the hierarchy.
	// See v1.Volume as an example
	inlined bool
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
	if b.inlined {
		// append to parent
		b.parent.SetAttributeValue(name, val)
	} else {
		b.hcl.Body().SetAttributeValue(name, val)
	}
}
