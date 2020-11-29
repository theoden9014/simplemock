package simplemock

import (
	"errors"
	"fmt"
	"go/types"
	"io"
	"reflect"
)

type SimpleMock struct {
	name      string
	interFace *types.Interface

	structGenerator *Struct
	funcGenerators  []*Func
}

func NewSimpleMock(name string, interFace *types.Interface) (*SimpleMock, error) {
	structGenerator := NewStruct(name, FieldList{})
	var funcGenerators []*Func

	// all methods
	for i := 0; i < interFace.NumMethods(); i++ {
		method := interFace.Method(i)
		method.Name()
		sig := method.Type().(*types.Signature)
		mockFieldName := method.Name() + `Func`
		field := NewField(mockFieldName, sig)
		if err := structGenerator.AddField(field); err != nil {
			return nil, fmt.Errorf("add field to struct: %w", err)
		}

		params, err := NewFieldListFromType(sig.Params())
		if err != nil {
			return nil, fmt.Errorf("failed to generate fields from types.Signature.Params(): %w", err)
		}
		results, err := NewFieldListFromType(sig.Results())
		if err != nil {
			return nil, fmt.Errorf("failed to generate fields from types.Signature.Results(): %w", err)
		}
		funcGenerator := NewFunc(method.Name(), params, results, structGenerator, "m", sig.Variadic())
		funcGenerator.SetBlockWriter(func(fn *Func, w io.Writer) error {
			recvName := fn.RecvName()
			fmt.Fprintln(w, `if `+recvName+`.`+mockFieldName+` != nil {`)
			params := fn.Params()
			if fn.Variadic() {
				fmt.Fprintln(w, `return m.`+mockFieldName+params.Format(FormatInputParamsWithVariadic))
			} else {
				fmt.Fprintln(w, `return m.`+mockFieldName+params.Format(FormatInputParams))
			}
			fmt.Fprintln(w, `}`)
			fmt.Fprintln(w, results.Format(FormatReturnZeroValueResults))
			return nil
		})
		funcGenerators = append(funcGenerators, funcGenerator)
	}

	m := &SimpleMock{
		name:            name,
		interFace:       interFace,
		structGenerator: structGenerator,
		funcGenerators:  funcGenerators,
	}
	return m, nil
}

func (m *SimpleMock) Name() string {
	return m.name
}

func (m *SimpleMock) WriteTo(w io.Writer) error {
	if err := m.structGenerator.WriteTo(w); err != nil {
		return fmt.Errorf("generate struct: %w", err)
	}
	for _, fg := range m.funcGenerators {
		fmt.Fprintln(w)
		if err := fg.WriteTo(w); err != nil {
			return fmt.Errorf("generate func: %w", err)
		}
	}
	return nil
}

type Struct struct {
	name   string
	fields FieldList
}

func NewStruct(name string, fields FieldList) *Struct {
	s := &Struct{name: name, fields: fields}
	return s
}

func (s *Struct) Name() string {
	return s.name
}

func (s *Struct) AddField(field *Field) error {
	s.fields.Add(field)
	if err := s.fields.Validate(); err != nil {
		return err
	}
	return nil
}

func (s *Struct) FieldList() FieldList {
	return s.fields
}

func (s *Struct) Type() *types.Struct {
	var fields []*types.Var
	//var tags []string
	for _, f := range s.fields {
		fields = append(fields, types.NewVar(0, nil, f.name, f.Type()))
	}
	return types.NewStruct(fields, nil)
}

func (s *Struct) WriteTo(w io.Writer) error {
	fmt.Fprintln(w, `type `+s.Name()+` struct {`)
	for _, field := range s.fields {
		fmt.Fprintln(w, field.String())
	}
	fmt.Fprintln(w, `}`)

	return nil
}

type Field struct {
	name string
	typ  types.Type
	tag  reflect.StructTag
}

func NewField(name string, typ types.Type) *Field {
	return &Field{name: name, typ: typ}
}

func (f *Field) SetTag(tag reflect.StructTag) {
	f.tag = tag
}

func (f *Field) Name() string {
	return f.name
}

// String is return "$var_name $type"
func (f *Field) String() string {
	if f.Name() == "" {
		return TypeString(f.Type())
	}
	return fmt.Sprintf("%s %s", f.name, TypeString(f.Type()))
}

func (f *Field) Type() types.Type {
	return f.typ
}

type FieldList []*Field

func NewFieldListFromType(t types.Type) (FieldList, error) {
	switch t := t.(type) {
	case *types.Tuple:
		fl := FieldList{}
		for i := 0; i < t.Len(); i++ {
			val := t.At(i)
			field := NewField(val.Name(), val.Type())
			fl.Add(field)
		}

		return fl, nil
	default:
		return nil, errors.New("not support type")
	}
}

func (fl FieldList) Validate() error {
	var checker map[string]bool
	for i := 0; i < fl.Len(); i++ {
		field := fl.At(i)
		fieldName := field.Name()
		if none := checker[fieldName]; none {
			return errors.New("there is a field with the same name")
		}
	}
	return nil
}

func (fl FieldList) String() (output string) {
	for i := 0; i < fl.Len(); i++ {
		output += fl[i].String()
		if i != fl.Len()-1 {
			output += `, `
		}
	}
	return output
}

func (fl *FieldList) Add(field *Field) {
	*fl = append(*fl, field)
}

func (fl FieldList) At(i int) *Field {
	return fl[i]
}

func (fl FieldList) Len() int {
	return len(fl)
}

func (fl FieldList) Swap(i, j int) {
	fl[i], fl[j] = fl[j], fl[i]
}

func (fl FieldList) Less(i, j int) bool {
	return fl[i].Name() > fl[j].Name()
}

type fieldListFormatter func(FieldList) string

// Format is translates into Go code that is available to each user by formatter.
func (fl FieldList) Format(formatter fieldListFormatter) (output string) {
	return formatter(fl)
}

type Func struct {
	name          string
	params        FieldList
	results       FieldList
	receiver      *Struct // todo: *Struct to abstract (type interface)
	receiverName  string
	valueReceiver bool
	blockWriter   func(*Func, io.Writer) error
	variadic      bool
}

func NewFunc(name string, params FieldList, results FieldList, receiver *Struct, receiverName string, variadic bool) *Func {
	fn := &Func{name: name, params: params, results: results, receiver: receiver, receiverName: receiverName, variadic: variadic}
	return fn
}

func (fn *Func) Name() string {
	return fn.name
}

func (fn *Func) Recv() *Struct {
	return fn.receiver
}

func (fn *Func) RecvName() string {
	return fn.receiverName
}

func (fn *Func) SetBlockWriter(f func(*Func, io.Writer) error) {
	fn.blockWriter = f
}

func (fn *Func) ValueReceiver() {
	fn.valueReceiver = true
}

func (fn *Func) Params() FieldList {
	return fn.params
}

func (fn *Func) Results() FieldList {
	return fn.results
}

func (fn *Func) Variadic() bool {
	return fn.variadic
}

func (fn *Func) WriteTo(w io.Writer) error {
	// not support non receiver
	if fn.receiver == nil {
		return errors.New("(t.b.d) implement if non receiver in Func.WriteTo")
	}
	recvType := fn.Recv().Name()
	if !fn.valueReceiver {
		recvType = `*` + recvType
	}

	var beforeResultsSpace string
	if fn.results.Len() != 0 {
		beforeResultsSpace += " "
	}
	if fn.variadic {
		last := fn.params.At(fn.params.Len() - 1)
		_, ok := last.Type().(*types.Slice)
		if !ok {
			return errors.New("variadic argument was expected but the last element is not sliced")
		}
		fn.params.Format(FormatDeclarativeParamsWithVariadic)
		fmt.Fprintln(w, `func (`+fn.RecvName()+` `+recvType+`)`+` `+fn.Name()+fn.params.Format(FormatDeclarativeParamsWithVariadic)+beforeResultsSpace+fn.results.Format(FormatDeclarativeResults)+` {`)
	} else {
		fmt.Fprintln(w, `func (`+fn.RecvName()+` `+recvType+`)`+` `+fn.Name()+fn.params.Format(FormatDeclarativeParams)+beforeResultsSpace+fn.results.Format(FormatDeclarativeResults)+` {`)
	}

	if fn.blockWriter != nil {
		if err := fn.blockWriter(fn, w); err != nil {
			return fmt.Errorf("generator block in func: %w", err)
		}
	}
	fmt.Fprintln(w, `}`)

	return nil
}

func FormatReturnZeroValueResults(fieldList FieldList) (output string) {
	output += "return"
	for i := 0; i < fieldList.Len(); i++ {
		output += " "
		field := fieldList.At(i)
		output += TypeZeroValue(field.Type())
		if i != fieldList.Len()-1 {
			output += ","
		}
	}

	return output
}

func FormatInputParams(fieldList FieldList) (output string) {
	if fieldList.Len() == 0 {
		return "()"
	}

	output += "("

	for i := 0; i < fieldList.Len(); i++ {
		field := fieldList.At(i)
		output += field.Name()
		if i < fieldList.Len()-1 {
			output += ", "
		}
	}

	output += ")"
	return output
}

func FormatInputParamsWithVariadic(fieldList FieldList) (output string) {
	if fieldList.Len() == 0 {
		return "()"
	}

	output += "("

	for i := 0; i < fieldList.Len(); i++ {
		field := fieldList.At(i)
		if i < fieldList.Len()-1 {
			output += field.Name()
			output += ", "
		} else { // last element
			last := field
			_, ok := last.Type().(*types.Slice)
			if !ok {
				output += field.Name()
			} else {
				output += field.Name() + "..."
			}
		}
	}

	output += ")"
	return output
}

func FormatDeclarativeParams(fieldList FieldList) (output string) {
	if fieldList.Len() == 0 {
		return "()"
	}

	output += "("

	for i := 0; i < fieldList.Len(); i++ {
		field := fieldList.At(i)
		output += field.String()
		if i < fieldList.Len()-1 {
			output += ", "
		}
	}

	output += ")"
	return output

}

func FormatDeclarativeParamsWithVariadic(fieldList FieldList) (output string) {
	output += "("

	for i := 0; i < fieldList.Len(); i++ {
		field := fieldList.At(i)
		if i < fieldList.Len()-1 {
			output += field.String()
			output += ", "
		} else { // last element
			last := field
			slice, ok := last.Type().(*types.Slice)
			if !ok {
				output += field.String()
			} else {
				elem := slice.Elem()
				output += field.Name() + " "
				output += "..." + TypeString(elem)
			}
		}
	}

	output += ")"
	return output
}

func FormatDeclarativeResults(fieldList FieldList) (output string) {
	if fieldList.Len() == 0 {
		return ""
	}

	if fieldList.Len() == 1 && fieldList.At(0).Name() == "" {
		return fieldList.At(0).String()
	}

	output += "("

	for i := 0; i < fieldList.Len(); i++ {
		field := fieldList.At(i)
		output += field.String()
		if i < fieldList.Len()-1 {
			output += ", "
		}
	}

	output += ")"
	return output
}
