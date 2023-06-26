package filestore

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertEqualFileContent сравнивает контент на идентичность(в рамках тестов, testify).
func AssertEqualFileContent(t *testing.T, expectedFile string, gotFile string) {
	// file with expected content
	require.FileExists(t, expectedFile)
	wantContent, err := os.ReadFile(expectedFile)
	require.NoError(t, err)

	// file with got content
	require.FileExists(t, expectedFile)
	gotContent, err := os.ReadFile(gotFile)
	require.NoError(t, err)
	assert.Equal(t, string(wantContent), string(gotContent))
}
