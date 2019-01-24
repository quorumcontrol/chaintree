package javascript

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	err := Run()
	require.Nil(t, err)
}
