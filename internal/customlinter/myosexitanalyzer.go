package customlinter

import (
	"go/ast"
	"golang.org/x/tools/go/analysis"
	"strings"
)

// MyOSExitAnalyzer анализатор, проверяющий наличие os.Exit в main-фунции main-пакета
var MyOSExitAnalyzer = &analysis.Analyzer{
	Name: "osExitAnalyzer",
	Doc:  "check if os.exit used in function 'main' of package 'main' ",
	Run:  osExitAnalyzerFunc,
}

// run функция для MyOSExitAnalyzer
func osExitAnalyzerFunc(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		// для пропуска сгенерированного go-test
		filename := pass.Fset.Position(file.Pos()).Filename
		if !strings.HasSuffix(filename, ".go") {
			continue
		}
		// если pkg не 'main' - пропускаем
		if packageName := file.Name.Name; packageName != "main" {
			continue
		}

		// функцией ast.Inspect проходим по всем узлам AST
		ast.Inspect(file, func(node ast.Node) bool {
			if funcDecl, ok := node.(*ast.FuncDecl); ok {
				if funcDecl.Name.Name == "main" {
					isOSExitUsed(pass, funcDecl.Body)
					// после проверки функции main - завершаем проход по AST файла
					return false
				}
			}
			return true
		})
	}
	return nil, nil
}

// в теле найденной функции main-функции main-пакета проверяет наличие os.Exit.
func isOSExitUsed(pass *analysis.Pass, mainFuncNode ast.Node) {
	ast.Inspect(mainFuncNode, func(node ast.Node) bool {
		// если найден вызов функции
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}

		// если этот вызов функции составной(пакет.функция())
		selCall, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		// если вызов функции состоит из 2х частей(точнее, левая часть это идентификатор)
		pkgIdent, ok := selCall.X.(*ast.Ident)
		if !ok {
			return true
		}

		// если левая часть вызова == os и правая часть == Exit - сообщить о "ошибке"
		if hasError := pkgIdent.Name == "os" && selCall.Sel.Name == "Exit"; hasError {
			pass.Reportf(selCall.Pos(), "os.Exit used in function 'main' of package 'main'")
		}
		return true
	})
}
