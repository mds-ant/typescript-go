package ast

import (
	"sync/atomic"

	"github.com/microsoft/typescript-go/internal/core"
)

type (
	NodeId   uint64
	SymbolId uint64
)

// Atomic ids

var (
	nextNodeId   atomic.Uint64
	nextSymbolId atomic.Uint64
)

func GetNodeId(node *Node) NodeId {
	return NodeId(getId(&node.id, &nextNodeId))
}

func GetSymbolId(symbol *Symbol) SymbolId {
	return SymbolId(getId(&symbol.id, &nextSymbolId))
}

func getId(id *atomic.Uint64, counter *atomic.Uint64) uint64 {
	value := id.Load()
	if value == 0 {
		// Worst case, we burn a few ids if we have to CAS.
		value = counter.Add(1)
		if !id.CompareAndSwap(0, value) {
			value = id.Load()
		}
	}
	return value
}

// Number of ids an IdAllocator reserves from a global counter at a time. A multiple of the paged link
// store page size so that pages never mix ids from different allocators.
const idBlockSize = 16 * core.LinkStorePageSize

// IdAllocator assigns node and symbol ids from contiguous blocks reserved from the global counters.
// Ids assigned through the same allocator cluster together, which keeps the pages of link stores keyed
// by those ids dense. An IdAllocator must not be used concurrently.
type IdAllocator struct {
	nodeIds   idBlock
	symbolIds idBlock
}

func (a *IdAllocator) GetNodeId(node *Node) NodeId {
	id := node.id.Load()
	if id == 0 {
		id = a.nodeIds.assign(&node.id, &nextNodeId)
	}
	return NodeId(id)
}

func (a *IdAllocator) GetSymbolId(symbol *Symbol) SymbolId {
	id := symbol.id.Load()
	if id == 0 {
		id = a.symbolIds.assign(&symbol.id, &nextSymbolId)
	}
	return SymbolId(id)
}

type idBlock struct {
	next uint64
	end  uint64
}

func (b *idBlock) assign(id *atomic.Uint64, counter *atomic.Uint64) uint64 {
	if b.next == b.end {
		b.reserve(counter)
	}
	b.next++
	if !id.CompareAndSwap(0, b.next) {
		b.next--
		return id.Load()
	}
	return b.next
}

// reserve claims a page-aligned block of ids from counter.
func (b *idBlock) reserve(counter *atomic.Uint64) {
	for {
		last := counter.Load()
		first := (last/core.LinkStorePageSize + 1) * core.LinkStorePageSize
		if counter.CompareAndSwap(last, first+idBlockSize-1) {
			b.next = first - 1
			b.end = first + idBlockSize - 1
			return
		}
	}
}
