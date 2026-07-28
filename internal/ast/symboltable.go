package ast

import (
	"iter"
	"maps"
	"slices"
	"strings"
)

// A SymbolTable maps names to symbols. It is used through *SymbolTable: a nil table is an
// empty read-only table, so Get, Len, All, Values, Delete and Clone accept a nil receiver.
// Stored symbols are never nil, and a symbol's Name must not change while it is stored in a
// table.
//
// Iteration is deterministic: entries come in key order, shortest key first and then in
// ascending byte order. Insertion order is not tracked; consumers that need a semantic order
// (such as declaration order) must sort. A table must not be mutated while it is iterated.
//
// A table is stored as a name-sorted []*Symbol (the key of an entry is its symbol's Name),
// which is several times smaller than a Go map for the small tables that dominate. It falls
// back to a Go map when a symbol is stored under a key other than its name.
//
// A SymbolTable is either in sorted form (m == nil), where symbols is sorted by name and each
// entry's key is its symbol's Name, or in map form (m != nil), where symbols is nil. Tables
// are held by pointer (*SymbolTable) and must not be copied by value.
type SymbolTable struct {
	symbols []*Symbol          // sorted form; nil in map form
	m       map[string]*Symbol // map form; nil in sorted form
}

func NewSymbolTable() *SymbolTable {
	return &SymbolTable{}
}

func NewSymbolTableWithCapacity(capacity int) *SymbolTable {
	return &SymbolTable{symbols: make([]*Symbol, 0, capacity)}
}

// compareKeys orders keys shortest first, then bytewise; comparing lengths first makes most
// binary search probes free of a string data comparison.
func compareKeys(a string, b string) int {
	if len(a) != len(b) {
		return len(a) - len(b)
	}
	return strings.Compare(a, b)
}

// find returns the index in the sorted form of the entry with the given name, or the index
// at which it would be inserted, and whether it was found.
func (t *SymbolTable) find(name string) (int, bool) {
	symbols := t.symbols
	lo, hi := 0, len(symbols)
	for lo < hi {
		mid := int(uint(lo+hi) >> 1)
		if compareKeys(symbols[mid].Name, name) < 0 {
			lo = mid + 1
		} else {
			hi = mid
		}
	}
	return lo, lo < len(symbols) && symbols[lo].Name == name
}

// Get returns the symbol stored under the given name, or nil. Small sorted tables are scanned
// linearly: an equality test rejects on length before comparing bytes, which beats a binary
// search's compares.
func (t *SymbolTable) Get(name string) *Symbol {
	if t == nil {
		return nil
	}
	if t.m != nil {
		return t.m[name]
	}
	if len(t.symbols) <= 8 {
		for _, symbol := range t.symbols {
			if symbol.Name == name {
				return symbol
			}
		}
		return nil
	}
	if i, ok := t.find(name); ok {
		return t.symbols[i]
	}
	return nil
}

// Set stores symbol under name. The table must not be nil (see GetSymbolTable) and symbol must
// not be nil.
func (t *SymbolTable) Set(name string, symbol *Symbol) {
	if t.m != nil {
		t.m[name] = symbol
		return
	}
	if name != symbol.Name {
		t.toMap()
		t.m[name] = symbol
		return
	}
	n := len(t.symbols)
	if n == 0 || compareKeys(t.symbols[n-1].Name, name) < 0 {
		// Fast path: appending in name order, the common case when copying or instantiating a table.
		if t.symbols == nil {
			t.symbols = make([]*Symbol, 0, 4) // most tables are tiny; skip the 1, 2, 4 growth steps
		}
		t.symbols = append(t.symbols, symbol)
		return
	}
	if i, ok := t.find(name); ok {
		t.symbols[i] = symbol
	} else {
		t.symbols = slices.Insert(t.symbols, i, symbol)
	}
}

// toMap converts a sorted-form table to map form.
func (t *SymbolTable) toMap() {
	m := make(map[string]*Symbol, len(t.symbols)+1)
	for _, symbol := range t.symbols {
		m[symbol.Name] = symbol
	}
	t.symbols = nil
	t.m = m
}

// Delete removes the entry for the given name, if present.
func (t *SymbolTable) Delete(name string) {
	if t == nil {
		return
	}
	if t.m != nil {
		delete(t.m, name)
		return
	}
	if i, ok := t.find(name); ok {
		t.symbols = slices.Delete(t.symbols, i, i+1)
	}
}

// Len returns the number of entries; a nil table is empty.
func (t *SymbolTable) Len() int {
	if t == nil {
		return 0
	}
	if t.m != nil {
		return len(t.m)
	}
	return len(t.symbols)
}

// All returns an iterator over (name, symbol) pairs in name order.
func (t *SymbolTable) All() iter.Seq2[string, *Symbol] {
	return func(yield func(string, *Symbol) bool) {
		if t == nil {
			return
		}
		if t.m != nil {
			for _, e := range t.sortedEntries() {
				if !yield(e.name, e.symbol) {
					return
				}
			}
			return
		}
		for _, symbol := range t.symbols {
			if !yield(symbol.Name, symbol) {
				return
			}
		}
	}
}

// Values returns an iterator over the symbols in name order.
func (t *SymbolTable) Values() iter.Seq[*Symbol] {
	return func(yield func(*Symbol) bool) {
		for _, symbol := range t.All() {
			if !yield(symbol) {
				return
			}
		}
	}
}

type symbolTableEntry struct {
	name   string
	symbol *Symbol
}

// sortedEntries returns the entries of a map-form table in name order.
func (t *SymbolTable) sortedEntries() []symbolTableEntry {
	entries := make([]symbolTableEntry, 0, len(t.m))
	for name, symbol := range t.m {
		entries = append(entries, symbolTableEntry{name, symbol})
	}
	slices.SortFunc(entries, func(a symbolTableEntry, b symbolTableEntry) int { return compareKeys(a.name, b.name) })
	return entries
}

// Clone returns a shallow copy of the table; a nil table clones to nil.
func (t *SymbolTable) Clone() *SymbolTable {
	if t == nil {
		return nil
	}
	if t.m != nil {
		return &SymbolTable{m: maps.Clone(t.m)}
	}
	return &SymbolTable{symbols: slices.Clone(t.symbols)}
}
