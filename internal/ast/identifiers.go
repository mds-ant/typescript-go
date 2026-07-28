package ast

import "github.com/microsoft/typescript-go/internal/collections"

// Identifiers returns the set of identifier spellings in the parsed file: the texts of all identifiers, private
// identifiers, and identifier-like literals (see isIdentifierLikeLiteral), including those in already parsed
// JSDoc. The set is computed on first use and cached; files derived from a parsed file by node factory updates
// share the parsed file's set.
func (file *SourceFile) Identifiers() *collections.Set[string] {
	source := file
	if source.identifierSource != nil {
		source = source.identifierSource
	}
	source.identifiersOnce.Do(func() {
		source.identifiers = collectIdentifiers(source)
	})
	return &source.identifiers
}

func collectIdentifiers(file *SourceFile) collections.Set[string] {
	var identifiers collections.Set[string]
	var visit Visitor
	visit = func(node *Node) bool {
		switch node.Kind {
		case KindIdentifier:
			if text := node.Text(); text != "" {
				identifiers.Add(text)
			}
		case KindPrivateIdentifier:
			identifiers.Add(node.Text())
		case KindStringLiteral, KindNoSubstitutionTemplateLiteral, KindNumericLiteral, KindBigIntLiteral:
			if isIdentifierLikeLiteral(node) {
				identifiers.Add(node.Text())
			}
		}
		node.ForEachChild(visit)
		return false
	}
	file.ForEachChild(visit)
	if file.hasLazyJSDoc {
		file.jsdocMu.RLock()
		defer file.jsdocMu.RUnlock()
	}
	for _, jsdocs := range file.jsdocCache {
		for _, jsdoc := range jsdocs {
			visit(jsdoc)
		}
	}
	return identifiers
}

// True if the literal is in a position where its text is used as a name.
func isIdentifierLikeLiteral(node *Node) bool {
	parent := node.Parent
	if parent == nil {
		return false
	}
	switch parent.Kind {
	case KindPropertyAssignment, KindPropertyDeclaration, KindPropertySignature, KindMethodDeclaration, KindMethodSignature,
		KindGetAccessor, KindSetAccessor, KindEnumMember, KindModuleDeclaration:
		return parent.Name() == node
	case KindBindingElement:
		return parent.AsBindingElement().PropertyName == node
	case KindImportDeclaration, KindJSImportDeclaration, KindExportDeclaration, KindJSDocImportTag:
		return parent.ModuleSpecifier() == node
	case KindExternalModuleReference:
		return parent.Expression() == node
	case KindElementAccessExpression:
		return parent.AsElementAccessExpression().ArgumentExpression == node && node.Kind != KindBigIntLiteral
	}
	return false
}
