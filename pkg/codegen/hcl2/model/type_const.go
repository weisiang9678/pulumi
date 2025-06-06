// Copyright 2016-2021, Pulumi Corporation.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

import (
	"strconv"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"

	"github.com/pulumi/pulumi/pkg/v3/codegen/hcl2/model/pretty"
	"github.com/pulumi/pulumi/pkg/v3/codegen/hcl2/syntax"
	"github.com/pulumi/pulumi/pkg/v3/util/gsync"
)

// ConstType represents a type that is a single constant value.
type ConstType struct {
	// Type is the underlying value type.
	Type Type
	// Value is the constant value.
	Value cty.Value

	cache *gsync.Map[Type, cacheEntry]
}

// NewConstType creates a new constant type with the given type and value.
func NewConstType(typ Type, value cty.Value) *ConstType {
	return &ConstType{Type: typ, Value: value, cache: &gsync.Map[Type, cacheEntry]{}}
}

func (t *ConstType) pretty(seenFormatters map[Type]pretty.Formatter) pretty.Formatter {
	if t.Value.IsNull() {
		return pretty.FromString("null")
	}
	if !t.Value.IsKnown() {
		return pretty.FromString("unknown")
	}
	switch t.Value.Type() {
	case cty.String:
		return pretty.FromString(strconv.Quote(t.Value.AsString()))
	case cty.Bool:
		return pretty.FromString(strconv.FormatBool(t.Value.True()))
	case cty.Number:
		return pretty.FromStringer(t.Value.AsBigFloat())
	}
	return pretty.FromStringer(t)
}

func (t *ConstType) Pretty() pretty.Formatter {
	seenFormatters := map[Type]pretty.Formatter{}
	return t.pretty(seenFormatters)
}

// SyntaxNode returns the syntax node for the type. This is always syntax.None.
func (*ConstType) SyntaxNode() hclsyntax.Node {
	return syntax.None
}

// Traverse attempts to traverse the type with the given traverser. The result is the traversal
// result of the underlying type.
func (t *ConstType) Traverse(traverser hcl.Traverser) (Traversable, hcl.Diagnostics) {
	return t.Type.Traverse(traverser)
}

// Equals returns true if this type has the same identity as the given type.
func (t *ConstType) Equals(other Type) bool {
	return t.equals(other, nil)
}

func (t *ConstType) equals(other Type, seen map[Type]struct{}) bool {
	if t == other {
		return true
	}

	otherConst, ok := other.(*ConstType)
	return ok && t.Value.RawEquals(otherConst.Value) && t.Type.equals(otherConst.Type, seen)
}

// AssignableFrom returns true if this type is assignable from the indicated source type. A const(value) is assignable
// from const(value).
func (t *ConstType) AssignableFrom(src Type) bool {
	return assignableFrom(t, src, func() bool {
		return false
	})
}

// ConversionFrom returns the kind of conversion (if any) that is possible from the source type to this type.
// The const type is only convertible from itself.
func (t *ConstType) ConversionFrom(src Type) ConversionKind {
	kind, _ := t.conversionFrom(src, false, nil)
	return kind
}

func (t *ConstType) conversionFrom(src Type, unifying bool, seen map[Type]struct{}) (ConversionKind, lazyDiagnostics) {
	if t.cache == nil {
		t.cache = &gsync.Map[Type, cacheEntry]{}
	}
	return conversionFrom(t, src, unifying, seen, t.cache, func() (ConversionKind, lazyDiagnostics) {
		if t.Type.ConversionFrom(src) != NoConversion {
			return UnsafeConversion, nil
		}
		return NoConversion, nil
	})
}

func (t *ConstType) String() string {
	return t.Value.GoString()
}

func (t *ConstType) string(_ map[Type]struct{}) string {
	return t.String()
}

func (t *ConstType) unify(other Type) (Type, ConversionKind) {
	return unify(t, other, func() (Type, ConversionKind) {
		return t, other.ConversionFrom(t)
	})
}

func (*ConstType) isType() {}

func IsConstType(t Type) bool {
	switch t := t.(type) {
	case *ConstType:
		return true
	case *UnionType:
		for _, t := range t.ElementTypes {
			_, ok := t.(*ConstType)
			if !ok {
				return false
			}
		}
		return true
	default:
		return false
	}
}
