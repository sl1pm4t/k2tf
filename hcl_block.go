package main

import (
	"github.com/hashicorp/hcl2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// hclBlock is a wrapper for hclwrite.Block that allows to tag some extra
// metadata to each block.
type hclBlock struct {
	name string

	// The parent hclBlock to this hclBlock
	parent *hclBlock

	// The wrapped HCL configuration block
	hcl *hclwrite.Block

	// hasValue means a child field of this block had a non-nil / empty value if
	// this is false when closeBlk() is called, the block won't be appended to
	// parent
	hasValue bool

	// inlined flags whether this block is "transparent" Some Structs in the
	// Kubernetes API structure are marked as "inline", meaning they don't create
	// a new block, and their child value is propagated up the hierarchy.
	// See v1.Volume as an example
	inlined bool
}

// we are closing a sub-block, write HCL to either:
// - parent Blocks HCL body in most cases
// - parent's parents HCL body if our parent is "inlined"
// - do nothing if the current hclBlock is inlined
func (b *hclBlock) AppendBlock(hcl *hclwrite.Block) {
	if b.inlined {
		// append to parent
		b.parent.AppendBlock(hcl)

	} else {
		b.hcl.Body().AppendBlock(hcl)

	}
}

func (b *hclBlock) SetAttributeValue(name string, val cty.Value) {
	if b.inlined {
		// append to parent
		b.parent.SetAttributeValue(name, val)
	} else {
		b.hcl.Body().SetAttributeValue(name, val)
	}
}
