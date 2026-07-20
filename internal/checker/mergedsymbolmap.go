package checker

import (
	"unsafe"

	"github.com/microsoft/typescript-go/internal/ast"
)

const mergedSymbolFilterLog2 = 18 // 32 KiB per checker

// mergedSymbolMap fronts the merged-symbols map with a bitset filter: almost no symbols are ever
// merged, so most lookups are rejected without a map probe. All writes go through add and entries
// are never deleted, so a negative filter answer is definitive.
type mergedSymbolMap struct {
	filter  [(1 << mergedSymbolFilterLog2) / 64]uint64
	entries map[*ast.Symbol]*ast.Symbol
}

func (m *mergedSymbolMap) add(source *ast.Symbol, target *ast.Symbol) {
	index := mergedSymbolFilterIndex(source)
	m.filter[index/64] |= 1 << (index % 64)
	if m.entries == nil {
		m.entries = make(map[*ast.Symbol]*ast.Symbol)
	}
	m.entries[source] = target
}

func (m *mergedSymbolMap) lookup(symbol *ast.Symbol) *ast.Symbol {
	index := mergedSymbolFilterIndex(symbol)
	if m.filter[index/64]&(1<<(index%64)) != 0 {
		return m.entries[symbol]
	}
	return nil
}

func mergedSymbolFilterIndex(symbol *ast.Symbol) uint64 {
	return (uint64(uintptr(unsafe.Pointer(symbol))) * 0x9E3779B97F4A7C15) >> (64 - mergedSymbolFilterLog2)
}
