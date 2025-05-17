package errors

import (
	"errors"
	"fmt"
	"runtime"
)

// Break can be used as a unique mark to communicate that an error represents a normal break point in a process with undefined end.
var Break = new(int)

// G allows an error slice of explicit type *T to pass as an error.
// Can be used to avoid unnecessary documentation + casting when all errors in a group are *T.
type G []*T

func (x G) Error() string {
	var p stringPrinter
	Traverse(&p, x)

	return string(p.bytes)
}

func (x G) Wrap() error {
	if len(x) == 0 {
		return nil
	}

	return x
}

// A Group can be used to pass multiple independent errors as a single error value.
type Group []error

// Add expands the Group, discaring nil values.
func (x *Group) Add(errs ...error) {
	for _, err := range errs {
		if err != nil {
			*x = append(*x, err)
		}
	}
}

func (x Group) Error() string {
	var p stringPrinter
	Traverse(&p, x)

	return string(p.bytes)
}

// Wrap returns the Group value, or nil if it is empty.
// Can be used to conveniently return a potential group of errors.
func (x Group) Wrap() error {
	if len(x) == 0 {
		return nil
	}

	return x
}

// A Line represents a line of code.
type Line struct {
	File   string
	Number int
}

// LineGet returns the Line of its callsite.
// Generally, this is the most convenient way to obtain an error trace, at a small runtime cost.
//
// NOTE this likely doesn't work in stripped builds
func LineGet() Line {
	_, file, line, _ := runtime.Caller(1)
	return Line{file, line}
}

type Printer interface {
	Group()
	GroupEnd()
	Print(*T)
	PrintError(error)
	Sub()
	SubEnd()
	Tail()
	TailEnd()
}

// T is the main error implementation introduced by this package. All of its fields can be treated as optional.
type T struct {
	// helps find location in code; usually the function name or a Line
	Trace any

	// helps understand; human readable
	Message string

	// extra information, such as local variables
	Info []any

	// helps caller make a decision at runtime
	Mark any

	// suberror that is being expanded upon
	Sub error

	// additional errors encountered in the follow-up code
	Tail []error

	// errors that tail to this one; helps prevent loops when formating
	Lead []*T
}

func Mark(mark any, err error) *T {
	return &T{
		Mark: mark,
		Sub:  err,
	}
}

// MarkBreak is a [Mark] variant that uses the unique Break value.
func MarkBreak(err error) *T {
	return Mark(Break, err)
}

func Message(msg string, err error) *T {
	return &T{
		Message: msg,
		Sub:     err,
	}
}

// Trace returns a T with given trace and suberror.
// It is meant as a convenience function for the most minimal error handling.
func Trace(trace any, err error) *T {
	return &T{
		Trace: trace,
		Sub:   err,
	}
}

// TraceLine is a [Trace] variant that uses its callsite Line as trace field, for even more minimalness.
func TraceLine(err error) *T {
	_, file, line, _ := runtime.Caller(1)
	return &T{
		Trace: Line{file, line},
		Sub:   err,
	}
}

func (x *T) Error() string {
	if x == nil {
		return ""
	}

	var p stringPrinter
	Traverse(&p, x)

	return string(p.bytes)
}

func (x *T) InfoAdd(v ...any) {
	x.Info = append(x.Info, v...)
}

// Link adds the given error as a Tail.
// If the error is of type *T, then this value is added to its Leads.
//
// Noop if the error is nil.
func (x *T) Link(err error) {
	if err == nil {
		return
	}

	x.Tail = append(x.Tail, err)

	if t, ok := err.(*T); ok {
		t.Lead = append(t.Lead, x)
	}
}

type stringPrinter struct {
	bytes       []byte
	indentBytes []byte
}

func (x *stringPrinter) Group() {
	x.line("[")
}

func (x *stringPrinter) GroupEnd() {
	x.line("]")
}

func (x *stringPrinter) Print(err *T) {
	if err.Message != "" {
		x.line(err.Message)
	}

	if err.Trace != nil {
		x.indent()
		x.string("Trace: ")
		x.any(err.Trace)
		x.newline()
	}

	if len(err.Info) > 0 {
		x.indent()
		x.string("Info: ")
		x.any(err.Info)
		x.newline()
	}
}

func (x *stringPrinter) PrintError(err error) {
	x.line(err.Error())
}

func (x *stringPrinter) Sub() {
	x.newline()
	x.indentBytes = append(x.indentBytes, ' ', ' ')
}

func (x *stringPrinter) SubEnd() {
	x.newline()
	x.indentBytes = x.indentBytes[:len(x.indentBytes)-2]
}

func (x *stringPrinter) Tail() {
	x.line("{")
}

func (x *stringPrinter) TailEnd() {
	x.line("}")
}

func (x *stringPrinter) any(v any) {
	x.bytes = fmt.Append(x.bytes, v)
}

func (x *stringPrinter) indent() {
	x.bytes = append(x.bytes, x.indentBytes...)
}

func (x *stringPrinter) line(s string) {
	x.indent()
	x.string(s)
	x.newline()
}

func (x *stringPrinter) newline() {
	x.bytes = append(x.bytes, '\n')
}

func (x *stringPrinter) string(s string) {
	x.bytes = append(x.bytes, s...)
}

// Simple returns a basic text error.
//
// Returned values will never compare true.
func Simple(s string) error {
	return errors.New(s)
}

func Traverse(p Printer, err error) {
	//TODO protect against Tail loops

	switch e := err.(type) {
	case *T:
		traverseT(p, e)
	case G:
		p.Group()
		for _, v := range e {
			traverseT(p, v)
		}
		p.GroupEnd()
	case Group:
		p.Group()
		for _, v := range e {
			Traverse(p, v)
		}
		p.GroupEnd()
	default:
		p.PrintError(e)
	}
}

func traverseT(p Printer, err *T) {
	p.Print(err)

	if err.Sub != nil {
		p.Sub()
		Traverse(p, err.Sub)
		p.SubEnd()
	}

	if len(err.Tail) > 0 {
		p.Tail()
		for _, v := range err.Tail {
			Traverse(p, v)
		}
		p.TailEnd()
	}
}
