package checker

import (
	"testing"

	"github.com/microsoft/typescript-go/internal/ast"
	"github.com/microsoft/typescript-go/internal/core"
	"github.com/microsoft/typescript-go/internal/parser"
	"github.com/microsoft/typescript-go/internal/testutil/fixtures"
	"github.com/microsoft/typescript-go/internal/tspath"
	"github.com/microsoft/typescript-go/internal/vfs/osvfs"
)

func collectGeneratorBodies(node *ast.Node, out *[]*ast.Node) {
	node.ForEachChild(func(child *ast.Node) bool {
		if ast.IsFunctionLikeDeclaration(child) && ast.GetFunctionFlags(child)&ast.FunctionFlagsGenerator != 0 {
			if body := child.Body(); body != nil {
				*out = append(*out, body)
			}
		}
		collectGeneratorBodies(child, out)
		return false
	})
}

func yieldExpressionVisitor(*ast.Node) bool { return false }

var forEachYieldExpressionSink bool

func BenchmarkForEachYieldExpression(b *testing.B) {
	for _, f := range fixtures.BenchFixtures {
		b.Run(f.Name(), func(b *testing.B) {
			f.SkipIfNotExist(b)

			fileName := tspath.GetNormalizedAbsolutePath(f.Path(), "/")
			path := tspath.ToPath(fileName, "/", osvfs.FS().UseCaseSensitiveFileNames())
			sourceText := f.ReadFile(b)
			scriptKind := core.GetScriptKindFromFileName(fileName)

			sf := parser.ParseSourceFile(ast.SourceFileParseOptions{
				FileName: fileName,
				Path:     path,
			}, sourceText, scriptKind)

			var bodies []*ast.Node
			collectGeneratorBodies(sf.AsNode(), &bodies)
			if len(bodies) == 0 {
				b.Skip("no generator bodies")
			}

			b.ReportAllocs()
			for b.Loop() {
				for _, body := range bodies {
					forEachYieldExpressionSink = forEachYieldExpression(body, yieldExpressionVisitor)
				}
			}
		})
	}
}
