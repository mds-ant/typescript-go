package checker

import (
	"encoding/binary"
	"slices"
	"strings"
	"testing"

	"github.com/zeebo/xxh3"
	"gotest.tools/v3/assert"
)

func TestCacheKeyEncoding(t *testing.T) {
	t.Parallel()

	inlineSize := len(keyBuilder{}.inline)
	types := func(n int) []*Type {
		result := make([]*Type, n)
		for i := range result {
			result[i] = &Type{id: TypeId(i*1000003 + 1)}
		}
		return result
	}
	uint32LE := func(v uint32) []byte { return binary.LittleEndian.AppendUint32(nil, v) }
	uint64LE := func(v uint64) []byte { return binary.LittleEndian.AppendUint64(nil, v) }
	typeList := func(list []*Type) []byte {
		bytes := uint64LE(uint64(len(list)))
		for _, typ := range list {
			bytes = append(bytes, uint32LE(uint32(typ.id))...)
		}
		return bytes
	}

	few := types(3)
	many := types(inlineSize/4 + 1)
	nearFull := types((inlineSize - 8 - 1) / 4)
	fitting := []string{strings.Repeat("a", inlineSize*3/4), strings.Repeat("b", inlineSize/2)}
	overflowing := strings.Repeat("x", inlineSize+1)

	tests := []struct {
		name  string
		key   CacheHashKey
		bytes []byte
	}{
		{
			name:  "empty type list",
			key:   getTypeListKey(nil),
			bytes: uint64LE(0),
		},
		{
			name:  "type list",
			key:   getTypeListKey(few),
			bytes: typeList(few),
		},
		{
			name:  "type list beyond the inline buffer",
			key:   getTypeListKey(many),
			bytes: typeList(many),
		},
		{
			name: "tuple elements",
			key: getTupleKey([]TupleElementInfo{
				{flags: ElementFlagsRequired},
				{flags: ElementFlagsOptional},
				{flags: ElementFlagsRest},
				{flags: ElementFlagsVariadic},
			}, true /*readonly*/),
			bytes: []byte("#?.*!"),
		},
		{
			name:  "tuple marker written at a full inline buffer",
			key:   getTupleKey(make([]TupleElementInfo, inlineSize), true /*readonly*/),
			bytes: append([]byte(strings.Repeat("*", inlineSize)), '!'),
		},
		{
			name:  "template lengths and texts straddling the inline buffer",
			key:   getTemplateTypeKey(fitting, nearFull),
			bytes: slices.Concat(typeList(nearFull), []byte("|"), uint64LE(uint64(len(fitting[0]))), uint64LE(uint64(len(fitting[1]))), []byte("|"), []byte(fitting[0]), []byte(fitting[1])),
		},
		{
			name:  "template text longer than the inline buffer",
			key:   getTemplateTypeKey([]string{overflowing}, few),
			bytes: slices.Concat(typeList(few), []byte("|"), uint64LE(uint64(len(overflowing))), []byte("|"), []byte(overflowing)),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, test.key, CacheHashKey(xxh3.Hash128(test.bytes)))
		})
	}
}

func TestRelationKeyBetweenPlainTypesIsPacked(t *testing.T) {
	t.Parallel()
	source, target := &Type{id: 5}, &Type{id: 7}
	key, constrained := getRelationKey(source, target, IntersectionStateSource, false /*isIdentity*/, false /*ignoreConstraints*/)
	assert.Assert(t, !constrained)
	assert.Equal(t, key, CacheHashKey{Hi: relationKeySimpleTag, Lo: uint64(source.id) | uint64(target.id)<<31 | uint64(IntersectionStateSource)<<62})
}
