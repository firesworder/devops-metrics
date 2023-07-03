package main

import (
	"github.com/firesworder/devopsmetrics/internal/customlinter"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(customlinter.GetAnalyzerList()...)
}
