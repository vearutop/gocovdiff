package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_parseCoverFunc(t *testing.T) {
	res, err := parseCoverFunc([]byte(`sample/bar.go:3:	Bar		71.4%
sample/foo.go:5:	foo		44.4%
total:			(statements)	56.2%
`))

	require.NoError(t, err)
	assert.Equal(t, []coverFunc{
		{filename: "sample/bar.go", funcname: "Bar", percent: "71.4"},
		{filename: "sample/foo.go", funcname: "foo", percent: "44.4"},
		{filename: "total", funcname: "(statements)", percent: "56.2"},
	}, res)
}
