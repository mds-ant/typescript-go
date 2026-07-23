package ast

import "github.com/microsoft/typescript-go/internal/core"

// Throwaway probe counters for shared-state accesses (TSGO_PROBE).
var (
	probeNodeIdCalls        = core.NewCounter("ast.GetNodeId.calls")
	probeNodeIdAssigns      = core.NewCounter("ast.GetNodeId.assigns")
	probeSymbolIdCalls      = core.NewCounter("ast.GetSymbolId.calls")
	probeSymbolIdAssigns    = core.NewCounter("ast.GetSymbolId.assigns")
	probeIdCASLosses        = core.NewCounter("ast.getId.casLosses")
	probeSubtreeFactsCalls  = core.NewCounter("ast.SubtreeFacts.calls")
	probeSubtreeFactsStores = core.NewCounter("ast.SubtreeFacts.computeStores")
	probeResolveJSDocCalls  = core.NewCounter("ast.SourceFile.resolveJSDoc.calls")
	probeResolveJSDocParses = core.NewCounter("ast.SourceFile.resolveJSDoc.parses")
	probeEagerJSDocLocked   = core.NewCounter("ast.Node.EagerJSDoc.lockedReads")
	probeECMALineMapCalls   = core.NewCounter("ast.SourceFile.ECMALineMap.calls")
	probeECMALineMapMakes   = core.NewCounter("ast.SourceFile.ECMALineMap.computes")
	probeFileDataCellCalls  = core.NewCounter("ast.getSourceFileDataCell.calls")
	probeDiagAdds           = core.NewCounter("ast.DiagnosticsCollection.Add.calls")
	probeDiagLookups        = core.NewCounter("ast.DiagnosticsCollection.Lookup.calls")
	probeSymbolTableMakes   = core.NewCounter("ast.GetSymbolTable.lazyMakes")
	probeGetNameTableOnce   = core.NewCounter("ast.SourceFile.GetNameTable.onceCalls")
)
