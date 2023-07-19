package customlinter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAnalyzerList(t *testing.T) {
	analyzerList := GetAnalyzerList()
	assert.Greater(t, len(analyzerList), 0, "analyzer list can not be empty")
}

func Test_getStaticCheckAnalyzers(t *testing.T) {
	// 90SA + 2S + 2ST
	assert.Equal(t, 90+2+2, len(getStaticCheckAnalyzers()))
}

func Test_isNameSliceContains(t *testing.T) {
	var wantResult, gotResult bool

	wantResult = true
	gotResult = isNameSliceContains([]string{"S1000", "SA1500", "ST0011"}, "SA1500")
	assert.Equal(t, wantResult, gotResult)

	wantResult = false
	gotResult = isNameSliceContains([]string{"S1000", "SA1500", "ST0011"}, "Q1010")
	assert.Equal(t, wantResult, gotResult)
}
