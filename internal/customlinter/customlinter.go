package customlinter

import (
	"github.com/gordonklaus/ineffassign/pkg/ineffassign"
	"github.com/sashamelentyev/usestdlibvars/pkg/analyzer"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/asmdecl"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/atomic"
	"golang.org/x/tools/go/analysis/passes/atomicalign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/analysis/passes/buildtag"
	"golang.org/x/tools/go/analysis/passes/cgocall"
	"golang.org/x/tools/go/analysis/passes/composite"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/ctrlflow"
	"golang.org/x/tools/go/analysis/passes/deepequalerrors"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/directive"
	"golang.org/x/tools/go/analysis/passes/errorsas"
	"golang.org/x/tools/go/analysis/passes/fieldalignment"
	"golang.org/x/tools/go/analysis/passes/findcall"
	"golang.org/x/tools/go/analysis/passes/framepointer"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/ifaceassert"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/reflectvaluecompare"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/shift"
	"golang.org/x/tools/go/analysis/passes/sigchanyzer"
	"golang.org/x/tools/go/analysis/passes/slog"
	"golang.org/x/tools/go/analysis/passes/sortslice"
	"golang.org/x/tools/go/analysis/passes/stdmethods"
	"golang.org/x/tools/go/analysis/passes/stringintconv"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/testinggoroutine"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unsafeptr"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"golang.org/x/tools/go/analysis/passes/usesgenerics"
	"honnef.co/go/tools/simple"
	"honnef.co/go/tools/staticcheck"
	"honnef.co/go/tools/stylecheck"
)

var SAnalyzersNames = []string{"S1005", "S1008"}
var STAnalyzersNames = []string{"ST1000", "ST1008"}

// StandardAnalyzers список стандартных анализаторов из analysis/passes
var StandardAnalyzers = []*analysis.Analyzer{
	asmdecl.Analyzer,
	assign.Analyzer,
	atomic.Analyzer,
	atomicalign.Analyzer,
	bools.Analyzer,
	buildssa.Analyzer,
	buildtag.Analyzer,
	cgocall.Analyzer,
	composite.Analyzer,
	copylock.Analyzer,
	ctrlflow.Analyzer,
	deepequalerrors.Analyzer,
	defers.Analyzer,
	directive.Analyzer,
	errorsas.Analyzer,
	fieldalignment.Analyzer,
	findcall.Analyzer,
	framepointer.Analyzer,
	httpresponse.Analyzer,
	ifaceassert.Analyzer,
	inspect.Analyzer,
	loopclosure.Analyzer,
	lostcancel.Analyzer,
	nilfunc.Analyzer,
	printf.Analyzer,
	reflectvaluecompare.Analyzer,
	shadow.Analyzer,
	shift.Analyzer,
	sigchanyzer.Analyzer,
	slog.Analyzer,
	sortslice.Analyzer,
	stdmethods.Analyzer,
	stringintconv.Analyzer,
	structtag.Analyzer,
	tests.Analyzer,
	testinggoroutine.Analyzer,
	timeformat.Analyzer,
	unmarshal.Analyzer,
	unreachable.Analyzer,
	unsafeptr.Analyzer,
	unusedresult.Analyzer,
	unusedwrite.Analyzer,
	usesgenerics.Analyzer,
}

// ChosenPublicAnalyzers список выбранных кастомных анализаторов
var ChosenPublicAnalyzers = []*analysis.Analyzer{
	ineffassign.Analyzer,
	analyzer.New(),
}

// StaticCheckAnalyzers анализаторы из static check(весь класс SA + анализаторы из SAnalyzersNames и STAnalyzersNames)
var StaticCheckAnalyzers = getStaticCheckAnalyzers()

// GetAnalyzerList возвращает список анализаторов(требуемых по заданию инкремента18).
func GetAnalyzerList() []*analysis.Analyzer {
	resultList := make([]*analysis.Analyzer, 0)
	resultList = append(resultList, StandardAnalyzers...)
	resultList = append(resultList, ChosenPublicAnalyzers...)
	resultList = append(resultList, StaticCheckAnalyzers...)
	resultList = append(resultList, MyOSExitAnalyzer)
	return resultList
}

// собирает анализаторы из staticcheck библиотеки.
// все SA(сейчас 90) + все S и ST из SAnalyzersNames и STAnalyzersNames соответственно.
func getStaticCheckAnalyzers() []*analysis.Analyzer {
	analyzerList := make([]*analysis.Analyzer, 0)

	// Все SA анализаторы
	for _, v := range staticcheck.Analyzers {
		analyzerList = append(analyzerList, v.Analyzer)
	}

	// simple анализаторы(класс S)
	for _, v := range simple.Analyzers {
		if isNameSliceContains(SAnalyzersNames, v.Analyzer.Name) {
			analyzerList = append(analyzerList, v.Analyzer)
		}
	}

	// stylecheck анализаторы(класс ST)
	for _, v := range stylecheck.Analyzers {
		if isNameSliceContains(STAnalyzersNames, v.Analyzer.Name) {
			analyzerList = append(analyzerList, v.Analyzer)
		}
	}

	return analyzerList
}

// проверяет, есть ли в слайсе строк - строка name. True, если есть.
func isNameSliceContains(namesSlice []string, name string) bool {
	for _, n := range namesSlice {
		if n == name {
			return true
		}
	}
	return false
}
