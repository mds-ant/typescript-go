package checker

import "github.com/microsoft/typescript-go/internal/ast"

// memberNameFilter rejects names that cannot be in a resolved member table, using a bitset over
// (name length, first byte). Rebuilt whenever the table's identity or size changes; member tables
// only ever grow in place, so a negative answer is definitive. Used for the global Object
// fallback, where nearly every looked-up name misses.
type memberNameFilter struct {
	source *StructuredType
	count  int
	bits   [64]uint64
}

func (f *memberNameFilter) mayContain(resolved *StructuredType, name string) bool {
	if f.source != resolved || f.count != len(resolved.members) {
		f.rebuild(resolved)
	}
	if len(name) == 0 {
		return true // never recorded; use the real lookup
	}
	return f.bits[len(name)%64]&(1<<(name[0]%64)) != 0
}

func (f *memberNameFilter) rebuild(resolved *StructuredType) {
	f.bits = [64]uint64{}
	for name := range resolved.members {
		if len(name) != 0 {
			f.bits[len(name)%64] |= 1 << (name[0] % 64)
		}
	}
	f.source = resolved
	f.count = len(resolved.members)
}

// getPropertyOfObjectType(c.globalObjectType, name), with the member-name filter in front.
func (c *Checker) getPropertyOfGlobalObjectType(name string) *ast.Symbol {
	t := c.globalObjectType
	if t.flags&TypeFlagsObject != 0 && !c.globalObjectMemberNames.mayContain(c.resolveStructuredTypeMembers(t), name) {
		return nil
	}
	return c.getPropertyOfObjectType(t, name)
}
