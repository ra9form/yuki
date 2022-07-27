// Code generated by protoc-gen-goyuki, but you can (must) modify it.
// source: strings.proto

package strings

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	desc "github.com/utrack/yuki/integration/go_package_separate_impl_exists/pkg/strings"
)

func TestStringsImplementation_ToLower(t *testing.T) {
	api := NewStrings()
	res, err := api.ToLower(context.Background(), &desc.String{Str: "1"})

	require.Nil(t, err)
	require.NotNil(t, res)
	require.Equal(t, "1", res.GetStr())
}
