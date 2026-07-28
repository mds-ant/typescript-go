package ast

import (
	"slices"
	"strconv"
	"testing"
	"unsafe"

	"gotest.tools/v3/assert"
)

func makeSymbol(name string) *Symbol {
	return &Symbol{Name: name}
}

func collectNames(t *SymbolTable) []string {
	var names []string
	for name := range t.All() {
		names = append(names, name)
	}
	return names
}

func collectValueNames(t *SymbolTable) []string {
	var names []string
	for symbol := range t.Values() {
		names = append(names, symbol.Name)
	}
	return names
}

func TestSymbolTableSetGetDelete(t *testing.T) {
	t.Parallel()
	table := NewSymbolTable()
	a, b, c := makeSymbol("alpha"), makeSymbol("b"), makeSymbol("gamma")
	table.Set(a.Name, a)
	table.Set(b.Name, b)
	table.Set(c.Name, c)
	assert.Equal(t, table.Len(), 3)
	assert.Equal(t, table.Get("alpha"), a)
	assert.Equal(t, table.Get("b"), b)
	assert.Equal(t, table.Get("gamma"), c)
	assert.Assert(t, table.Get("missing") == nil)
	b2 := makeSymbol("b")
	table.Set("b", b2)
	assert.Equal(t, table.Len(), 3)
	assert.Equal(t, table.Get("b"), b2)
	table.Delete("b")
	assert.Equal(t, table.Len(), 2)
	assert.Assert(t, table.Get("b") == nil)
	table.Delete("missing")
	assert.Equal(t, table.Len(), 2)
}

func TestSymbolTableNilIsEmpty(t *testing.T) {
	t.Parallel()
	var table *SymbolTable
	assert.Equal(t, table.Len(), 0)
	assert.Assert(t, table.Get("x") == nil)
	assert.Assert(t, table.Clone() == nil)
	table.Delete("x")
	assert.Equal(t, len(collectNames(table)), 0)
	assert.Equal(t, len(collectValueNames(table)), 0)
}

func TestSymbolTableIteratesShortestNameFirstThenBytewise(t *testing.T) {
	t.Parallel()
	names := []string{"zeta", "\xFEcall", "alpha", "b", "ab", "aa", "prototype", "constructor"}
	expected := []string{"b", "aa", "ab", "zeta", "alpha", "\xFEcall", "prototype", "constructor"}
	table := NewSymbolTable()
	for _, name := range names {
		table.Set(name, makeSymbol(name))
	}
	assert.DeepEqual(t, collectNames(table), expected)
	assert.DeepEqual(t, collectValueNames(table), expected)
	for _, name := range names {
		assert.Equal(t, table.Get(name).Name, name)
	}
}

func TestSymbolTableLargeBuiltInNameOrder(t *testing.T) {
	t.Parallel()
	names := make([]string, 5000)
	for i := range names {
		names[i] = "sym" + strconv.Itoa(i)
	}
	slices.SortFunc(names, compareKeys)
	table := NewSymbolTableWithCapacity(len(names))
	for _, name := range names {
		table.Set(name, makeSymbol(name))
	}
	assert.Equal(t, table.Len(), len(names))
	assert.DeepEqual(t, collectNames(table), names)
	for _, name := range names {
		assert.Equal(t, table.Get(name).Name, name)
	}
	assert.Assert(t, table.Get("sym5000") == nil)
	assert.Assert(t, table.Get("sy") == nil)
}

func TestSymbolTableLargeBuiltOutOfNameOrder(t *testing.T) {
	t.Parallel()
	const n = 5000
	table := NewSymbolTable()
	for i := n - 1; i >= 0; i-- {
		name := strconv.Itoa(i) + "sym"
		table.Set(name, makeSymbol(name))
	}
	assert.Equal(t, table.Len(), n)
	for i := range n {
		assert.Equal(t, table.Get(strconv.Itoa(i)+"sym").Name, strconv.Itoa(i)+"sym")
	}
	assert.Assert(t, table.Get(strconv.Itoa(n)+"sym") == nil)
	names := collectNames(table)
	assert.Equal(t, len(names), n)
	assert.Assert(t, slices.IsSortedFunc(names, compareKeys))
	assert.DeepEqual(t, collectValueNames(table), names)
	replacement := makeSymbol("100sym")
	table.Set(replacement.Name, replacement)
	assert.Equal(t, table.Len(), n)
	assert.Equal(t, table.Get("100sym"), replacement)
	table.Delete("100sym")
	assert.Equal(t, table.Len(), n-1)
	assert.Assert(t, table.Get("100sym") == nil)
}

func TestSymbolTableKeyDifferentFromName(t *testing.T) {
	t.Parallel()
	table := NewSymbolTable()
	x, y := makeSymbol("x"), makeSymbol("y")
	table.Set("x", x)
	table.Set("alias", y)
	assert.Equal(t, table.Get("x"), x)
	assert.Equal(t, table.Get("alias"), y)
	assert.Assert(t, table.Get("y") == nil)
	assert.DeepEqual(t, collectNames(table), []string{"x", "alias"})
	assert.DeepEqual(t, collectValueNames(table), []string{"x", "y"})
	clone := table.Clone()
	table.Delete("alias")
	assert.Assert(t, table.Get("alias") == nil)
	assert.Equal(t, table.Len(), 1)
	assert.Equal(t, clone.Get("alias"), y)
	assert.Equal(t, clone.Len(), 2)
}

func TestSymbolTableEnumerationOrderIsIndependentOfInsertionOrder(t *testing.T) {
	t.Parallel()
	const n = 3000
	names := make([]string, n)
	for i := range names {
		names[i] = strconv.Itoa(i*7919%n) + "sym" + strconv.Itoa(i%13)
	}
	forward := NewSymbolTable()
	for _, name := range names {
		forward.Set(name, makeSymbol(name))
	}
	backward := NewSymbolTable()
	for i := len(names) - 1; i >= 0; i-- {
		backward.Set(names[i], makeSymbol(names[i]))
	}
	assert.DeepEqual(t, collectNames(forward), collectNames(backward))
	sorted := slices.Clone(names)
	slices.SortFunc(sorted, compareKeys)
	assert.DeepEqual(t, collectNames(forward), sorted)
}

func TestSymbolTableCloneIsIndependent(t *testing.T) {
	t.Parallel()
	table := NewSymbolTable()
	a := makeSymbol("a")
	table.Set("a", a)
	clone := table.Clone()
	clone.Set("b", makeSymbol("b"))
	assert.Equal(t, table.Len(), 1)
	assert.Equal(t, clone.Len(), 2)
	assert.Equal(t, clone.Get("a"), a)
	table.Set("c", makeSymbol("c"))
	assert.Assert(t, clone.Get("c") == nil)
	assert.Equal(t, clone.Len(), 2)
}

func TestSymbolTableCloneOfLargeTableIsIndependent(t *testing.T) {
	t.Parallel()
	const n = 3000
	table := NewSymbolTable()
	for i := n - 1; i >= 0; i-- {
		name := strconv.Itoa(i) + "sym"
		table.Set(name, makeSymbol(name))
	}
	clone := table.Clone()
	assert.Equal(t, clone.Len(), n)
	assert.DeepEqual(t, collectNames(clone), collectNames(table))
	clone.Delete("1sym")
	table.Set("extra", makeSymbol("extra"))
	assert.Assert(t, table.Get("1sym") != nil)
	assert.Assert(t, clone.Get("extra") == nil)
	assert.Equal(t, table.Len(), n+1)
	assert.Equal(t, clone.Len(), n-1)
}

func TestSymbolTableHandlesShareTheirTable(t *testing.T) {
	t.Parallel()
	table := NewSymbolTable()
	other := table
	table.Set("a", makeSymbol("a"))
	assert.Equal(t, other.Len(), 1)
	assert.Assert(t, other.Get("a") != nil)
	for i := 2999; i >= 0; i-- {
		name := strconv.Itoa(i) + "sym"
		other.Set(name, makeSymbol(name))
	}
	assert.Equal(t, table.Len(), 3001)
	assert.Assert(t, table.Get("2999sym") != nil)
	table.Set("renamed", makeSymbol("target"))
	assert.Equal(t, other.Get("renamed").Name, "target")
}

func TestSymbolTableCostsFourWordsPlusOneWordPerEntry(t *testing.T) {
	t.Parallel()
	word := unsafe.Sizeof(uintptr(0))
	assert.Equal(t, unsafe.Sizeof(SymbolTable{}), 4*word)
	assert.Equal(t, unsafe.Sizeof((*Symbol)(nil)), word)
}
