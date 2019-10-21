package modeler_openapiv3

import (
	"fmt"
	"io"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	// "github.com/FuzzyMonkeyCo/monkey/pkg/runtime"
	"go.starlark.net/starlark"
)

// func init() {
// 	runtime.RegisterModeler("OpenAPIv3", (*oa3)(nil))
// }

var _ modeler.Interface = (*oa3)(nil)

type T = oa3

type oa3 struct {
	fm.Clt_Msg_Fuzz_Model_OpenAPIv3

	resetter resetter.Interface

	vald *validator

	tcap *tCapHTTP
}

// ToProto TODO
func (m *oa3) ToProto() *fm.Clt_Msg_Fuzz_Model {
	return &fm.Clt_Msg_Fuzz_Model{
		Model: &fm.Clt_Msg_Fuzz_Model_Openapiv3{
			&m.Clt_Msg_Fuzz_Model_OpenAPIv3,
		}}
}

// SetResetter TODO
func (m *oa3) SetResetter(sr resetter.Interface) { m.resetter = sr }

// GetResetter TODO
func (m *oa3) GetResetter() resetter.Interface { return m.resetter }

// Pretty TODO
func (m *oa3) Pretty(w io.Writer) (int, error) { return fmt.Fprintf(w, "%+v\n", m) }

func (m *oa3) NewFromKwargs(d starlark.StringDict) (modeler.Interface, *modeler.Error) {
	m = &oa3{}
	var (
		found              bool
		field              string
		file, host, hAuthz starlark.Value
	)

	field = "file"
	if file, found = d[field]; !found || file.Type() != "string" {
		e := modeler.NewError(field, "a string", file.Type())
		return nil, e
	}
	m.File = file.(starlark.String).GoString()

	field = "host"
	if host, found = d[field]; found && host.Type() != "string" {
		e := modeler.NewError(field, "a string", file.Type())
		return nil, e
	}
	if found {
		h := host.(starlark.String).GoString()
		m.Host = h
	}

	field = "header_authorization"
	if hAuthz, found = d[field]; found && hAuthz.Type() != "string" {
		e := modeler.NewError(field, "a string", file.Type())
		return nil, e
	}
	if found {
		authz := hAuthz.(starlark.String).GoString()
		m.HeaderAuthorization = authz
	}

	return m, nil
}

func (m *oa3) InputsCount() int {
	return m.vald.InputsCount()
}

func (m *oa3) FilterEndpoints(args []string) ([]eid, error) {
	return m.vald.FilterEndpoints(args)
}

func (m *oa3) ValidateAgainstSchema(absRef string, data []byte) error {
	return m.vald.ValidateAgainstSchema(absRef, data)
}

func (m *oa3) WriteAbsoluteReferences(w io.Writer) {
	m.vald.WriteAbsoluteReferences(w)
}