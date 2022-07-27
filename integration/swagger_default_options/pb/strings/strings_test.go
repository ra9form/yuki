package strings

import (
	"strings"
	"testing"

	"github.com/ra9form/yuki/transport"
	"github.com/ra9form/yuki/transport/swagger"
	"github.com/stretchr/testify/assert"
)

func TestSwaggerComments(t *testing.T) {
	so := assert.New(t)

	d := NewStringsServiceDesc(nil)
	desc := "some description here"
	d.Apply(transport.WithSwaggerOptions(swagger.WithDescription(desc)))

	sdef := string(d.SwaggerDef())

	so.True(strings.Contains(sdef, desc))
}
