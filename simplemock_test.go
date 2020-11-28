package simplemock

import (
	"bytes"
	"fmt"
	"go/types"
	"io"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/packages/packagestest"

	"github.com/google/go-cmp/cmp"
)

func TestStruct_WriteTo(t *testing.T) {
	type fields struct {
		name   string
		fields FieldList
	}
	tests := []struct {
		name    string
		fields  fields
		wantW   string
		wantErr bool
	}{
		{
			name: "happy path",
			fields: fields{
				name: "User",
				fields: FieldList{
					NewField("name", types.Typ[types.String]),
					NewField("id", types.Typ[types.Int64]),
				},
			},
			wantW: `type User struct {
name string
id int64
}
`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Struct{
				name:   tt.fields.name,
				fields: tt.fields.fields,
			}
			w := &bytes.Buffer{}
			err := s.WriteTo(w)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotW := w.String()
			if diff := cmp.Diff(tt.wantW, gotW); diff != "" {
				t.Errorf("WriteTo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFunc_WriteTo(t *testing.T) {
	type fields struct {
		name          string
		params        FieldList
		results       FieldList
		receiver      *Struct
		receiverName  string
		valueReceiver bool
		blockWriter   func(*Func, io.Writer) error
	}
	tests := []struct {
		name    string
		fields  fields
		wantW   string
		wantErr bool
	}{
		{
			name: "non results",
			fields: fields{
				name: "SetIDName",
				params: FieldList{
					NewField("id", types.Typ[types.Int64]),
					NewField("name", types.Typ[types.String])},
				results:       FieldList{},
				receiver:      NewStruct("User", FieldList{}),
				receiverName:  "u",
				valueReceiver: false,
				blockWriter:   nil,
			},
			wantW: `func (u *User) SetIDName(id int64, name string) {
}
`,
			wantErr: false,
		},
		{
			name: "single results",
			fields: fields{
				name: "SetIDName",
				params: FieldList{
					NewField("id", types.Typ[types.Int64]),
					NewField("name", types.Typ[types.String])},
				results: FieldList{
					NewField("", types.Typ[types.Bool])},
				receiver:      NewStruct("User", FieldList{}),
				receiverName:  "u",
				valueReceiver: false,
				blockWriter:   nil,
			},
			wantW: `func (u *User) SetIDName(id int64, name string) bool {
}
`,
			wantErr: false,
		},
		{
			name: "some results",
			fields: fields{
				name: "SetIDName",
				params: FieldList{
					NewField("id", types.Typ[types.Int64]),
					NewField("name", types.Typ[types.String])},
				results: FieldList{
					NewField("", types.Typ[types.Bool]),
					NewField("", types.Typ[types.Bool])},
				receiver:      NewStruct("User", FieldList{}),
				receiverName:  "u",
				valueReceiver: false,
				blockWriter:   nil,
			},
			wantW: `func (u *User) SetIDName(id int64, name string) (bool, bool) {
}
`,
			wantErr: false,
		},
		{
			name: "use blockWriter",
			fields: fields{
				name: "SetIDName",
				params: FieldList{
					NewField("id", types.Typ[types.Int64]),
					NewField("name", types.Typ[types.String])},
				results: FieldList{
					NewField("", types.Typ[types.Bool])},
				receiver:      NewStruct("User", FieldList{}),
				receiverName:  "u",
				valueReceiver: false,
				blockWriter: func(fn *Func, w io.Writer) error {
					fmt.Fprintln(w, fn.RecvName()+`.SetID(id)`)
					fmt.Fprintln(w, fn.RecvName()+`.SetName(name)`)
					fmt.Fprintln(w, "return true")
					return nil
				},
			},
			wantW: `func (u *User) SetIDName(id int64, name string) bool {
u.SetID(id)
u.SetName(name)
return true
}
`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := &Func{
				name:          tt.fields.name,
				params:        tt.fields.params,
				results:       tt.fields.results,
				receiver:      tt.fields.receiver,
				receiverName:  tt.fields.receiverName,
				valueReceiver: tt.fields.valueReceiver,
				blockWriter:   tt.fields.blockWriter,
			}
			w := &bytes.Buffer{}
			err := fn.WriteTo(w)
			if (err != nil) != tt.wantErr {
				t.Errorf("WriteTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotW := w.String()
			if diff := cmp.Diff(tt.wantW, gotW); diff != "" {
				t.Errorf("WriteTo() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestFieldList_Format(t *testing.T) {
	type args struct {
		formatter fieldListFormatter
	}
	tests := []struct {
		name       string
		fl         FieldList
		args       args
		wantOutput string
	}{
		{
			name:       "FormatZeroValueResults (zero)",
			fl:         FieldList{},
			args:       args{FormatReturnZeroValueResults},
			wantOutput: `return`,
		},
		{
			name: "FormatZeroValueResults (single)",
			fl: FieldList{
				NewField("arg1", types.Typ[types.Int64])},
			args:       args{FormatReturnZeroValueResults},
			wantOutput: `return 0`,
		},
		{
			name: "FormatZeroValueResults (any)",
			fl: FieldList{
				NewField("arg1", types.Typ[types.Int64]),
				NewField("arg2", types.Typ[types.String])},
			args:       args{FormatReturnZeroValueResults},
			wantOutput: `return 0, ""`,
		},
		{
			name:       "FormatInputParams (zero)",
			fl:         FieldList{},
			args:       args{FormatInputParams},
			wantOutput: `()`,
		},
		{
			name: "FormatInputParams (single)",
			fl: FieldList{
				NewField("arg1", types.Typ[types.Int64])},
			args:       args{FormatInputParams},
			wantOutput: `(arg1)`,
		},
		{
			name: "FormatInputParams (any)",
			fl: FieldList{
				NewField("arg1", types.Typ[types.Int64]),
				NewField("arg2", types.Typ[types.String])},
			args:       args{FormatInputParams},
			wantOutput: `(arg1, arg2)`,
		},
		{
			name:       "FormatDeclarativeParams (zero)",
			fl:         FieldList{},
			args:       args{FormatDeclarativeParams},
			wantOutput: `()`,
		},
		{
			name: "FormatDeclarativeParams (single)",
			fl: FieldList{
				NewField("arg1", types.Typ[types.Int64])},
			args:       args{FormatDeclarativeParams},
			wantOutput: `(arg1 int64)`,
		},
		{
			name: "FormatDeclarativeParams (any)",
			fl: FieldList{
				NewField("arg1", types.Typ[types.Int64]),
				NewField("arg2", types.Typ[types.String])},
			args:       args{FormatDeclarativeParams},
			wantOutput: `(arg1 int64, arg2 string)`,
		},
		{
			name:       "FormatDeclarativeResults (zero)",
			fl:         FieldList{},
			args:       args{FormatDeclarativeResults},
			wantOutput: ``,
		},
		{
			name: "FormatDeclarativeResults (single)",
			fl: FieldList{
				NewField("arg1", types.Typ[types.Int64])},
			args:       args{FormatDeclarativeResults},
			wantOutput: `(arg1 int64)`,
		},
		{
			name: "FormatDeclarativeResults (single) (no named field)",
			fl: FieldList{
				NewField("", types.Typ[types.Int64])},
			args:       args{FormatDeclarativeResults},
			wantOutput: `int64`,
		},
		{
			name: "FormatDeclarativeResults (any)",
			fl: FieldList{
				NewField("arg1", types.Typ[types.Int64]),
				NewField("arg2", types.Typ[types.String])},
			args:       args{FormatDeclarativeResults},
			wantOutput: `(arg1 int64, arg2 string)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOutput := tt.fl.Format(tt.args.formatter); gotOutput != tt.wantOutput {
				t.Errorf("Format() = %v, want %v", gotOutput, tt.wantOutput)
			}
		})
	}
}

func TestSimpleMock_WriteTo(t *testing.T) {
	tests := []struct {
		name    string
		pkgpath string
		src     string
		wantW   string
		wantErr bool
	}{
		{
			name:    "",
			pkgpath: "example.com/util",
			src: `package util
import "io"

type Buffer interface {
	io.Writer
	io.Reader
	Reset()
}
`,
			wantW: `type BufferMock struct {
ReadFunc func(p []byte) (n int, err error)
ResetFunc func()
WriteFunc func(p []byte) (n int, err error)
}

func (m *BufferMock) Read(p []byte) (n int, err error) {
if m.ReadFunc != nil {
return m.ReadFunc(p)
}
return 0, nil
}

func (m *BufferMock) Reset() {
if m.ResetFunc != nil {
return m.ResetFunc()
}
return
}

func (m *BufferMock) Write(p []byte) (n int, err error) {
if m.WriteFunc != nil {
return m.WriteFunc(p)
}
return 0, nil
}
`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exported := packagestest.Export(t, packagestest.Modules, []packagestest.Module{
				{
					Name: filepath.Dir(tt.pkgpath),
					Files: map[string]interface{}{
						filepath.Join(filepath.Base(tt.pkgpath), "x.go"): tt.src,
					},
				},
			})
			defer exported.Cleanup()
			exported.Config.Mode = packages.NeedName | packages.NeedCompiledGoFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo
			pkgs, err := packages.Load(exported.Config, tt.pkgpath)
			if err != nil {
				t.Fatal(err)
			}
			if len(pkgs) != 1 {
				t.Fatal("load package error")
			}
			pkg := pkgs[0]
			for _, f := range pkg.Syntax {
				err := walk(f, pkg.TypesInfo,  func(iface string, ifaceType *types.Interface, err error) error {
					if err != nil {
						t.Fatal(err)
					}
					mockname := iface+"Mock"
					mock, err := NewSimpleMock(mockname, ifaceType)
					if err != nil {
						t.Fatal(err)
					}

					w := &bytes.Buffer{}
					err = mock.WriteTo(w)
					if (err != nil) != tt.wantErr {
						t.Errorf("WriteTo() error = %v, wantErr %v", err, tt.wantErr)
					}
					gotW, wantW := w.String(), tt.wantW
					if diff := cmp.Diff(gotW, wantW); diff != "" {
						t.Errorf("WriteTo() mismatch (-want +got):\n%s", diff)
					}

					return nil
				})
				if err != nil {
					t.Fatal(err)
				}
			}

		})
	}
}
