package checker

import (
	"unsafe"

	"github.com/microsoft/typescript-go/internal/ast"
)

// mergedSymbolMap fronts the merged-symbols map with a bitset filter: almost no symbols are ever
// merged, so most lookups are rejected without a map probe. All writes go through add and entries
// are never deleted, so a negative filter answer is definitive.
type mergedSymbolMap struct {
	filter  mergedSymbolFilter
	entries map[*ast.Symbol]*ast.Symbol
}

func (m *mergedSymbolMap) add(source *ast.Symbol, target *ast.Symbol) {
	m.filter.add(source)
	if m.entries == nil {
		m.entries = make(map[*ast.Symbol]*ast.Symbol)
	}
	m.entries[source] = target
}

func (m *mergedSymbolMap) lookup(symbol *ast.Symbol) *ast.Symbol {
	if m.filter.mayContain(symbol) {
		return m.entries[symbol]
	}
	return nil
}

const mergedSymbolFilterLog2 = 18 // 32 KiB per checker

type mergedSymbolFilter [(1 << mergedSymbolFilterLog2) / 64]uint64

// Symbol addresses are stable (arena chunks never move), so the pointer is a valid key.
func mergedSymbolFilterIndex(symbol *ast.Symbol) uint64 {
	return (uint64(uintptr(unsafe.Pointer(symbol))) * 0x9E3779B97F4A7C15) >> (64 - mergedSymbolFilterLog2)
}

func (f *mergedSymbolFilter) add(symbol *ast.Symbol) {
	index := mergedSymbolFilterIndex(symbol)
	f[index/64] |= 1 << (index % 64)
}

func (f *mergedSymbolFilter) mayContain(symbol *ast.Symbol) bool {
	index := mergedSymbolFilterIndex(symbol)
	return f[index/64]&(1<<(index%64)) != 0
}
