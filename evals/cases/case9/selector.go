// inlined from: https://github.com/theory/jsonpath/blob/48554e0502cd108828c99e48ef90f56382bc7e93/spec/selector.go#L272
package jsonpath

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// stringWriter defines the interface for JSONPath objects to write string
// representations of themselves to a string buffer.
type stringWriter interface {
	fmt.Stringer
	// writeTo writes a string to buf.
	writeTo(buf *strings.Builder)
}

// Selector represents a single Selector in an RFC 9535 JSONPath query.
type Selector interface {
	stringWriter

	// Select selects values from current and/or root and returns them.
	Select(current, root any) []any

	// SelectLocated selects values from current and/or root and returns them
	// in [LocatedNode] values with their located normalized paths
	SelectLocated(current, root any, parent NormalizedPath) []*LocatedNode

	// isSingular returns true for selectors that can only return a single
	// value.
	isSingular() bool
}

// Name is a key name selector, e.g., .name or ["name"], as defined by [RFC
// 9535 Section 2.3.1]. Interfaces implemented:
//   - [Selector]
//   - [fmt.Stringer]
//   - [NormalSelector]
//
// [RFC 9535 Section 2.3.1]: https://www.rfc-editor.org/rfc/rfc9535.html#name-name-selector
type Name string

// isSingular returns true because Name selects a single value from an object.
// Defined by the [Selector] interface.
func (Name) isSingular() bool { return true }

// String returns the quoted string representation of n.
func (n Name) String() string {
	return strconv.Quote(string(n))
}

// writeTo writes a quoted string representation of i to buf. Defined by
// [stringWriter].
func (n Name) writeTo(buf *strings.Builder) {
	buf.WriteString(n.String())
}

// Select selects n from input and returns it as a single value in a slice.
// Returns an empty slice if input is not a map[string]any or if it does not
// contain n. Defined by the [Selector] interface.
func (n Name) Select(input, _ any) []any {
	if obj, ok := input.(map[string]any); ok {
		if val, ok := obj[string(n)]; ok {
			return []any{val}
		}
	}
	return make([]any, 0)
}

// SelectLocated selects n from input and returns it with its normalized path
// as a single [LocatedNode] in a slice. Returns an empty slice if input is
// not a map[string]any or if it does not contain n. Defined by the [Selector]
// interface.
func (n Name) SelectLocated(input, _ any, parent NormalizedPath) []*LocatedNode {
	if obj, ok := input.(map[string]any); ok {
		if val, ok := obj[string(n)]; ok {
			return []*LocatedNode{newLocatedNode(append(parent, n), val)}
		}
	}
	return make([]*LocatedNode, 0)
}

// writeNormalizedTo writes n to buf formatted as a [normalized path] element.
// Defined by [NormalSelector].
//
// [normalized path]: https://www.rfc-editor.org/rfc/rfc9535#section-2.7
func (n Name) writeNormalizedTo(buf *strings.Builder) {
	// https://www.rfc-editor.org/rfc/rfc9535#section-2.7
	buf.WriteString("['")
	for _, r := range string(n) {
		switch r {
		case '\b': //  b BS backspace U+0008
			buf.WriteString(`\b`)
		case '\f': // f FF form feed U+000C
			buf.WriteString(`\f`)
		case '\n': // n LF line feed U+000A
			buf.WriteString(`\n`)
		case '\r': // r CR carriage return U+000D
			buf.WriteString(`\r`)
		case '\t': // t HT horizontal tab U+0009
			buf.WriteString(`\t`)
		case '\'': // ' apostrophe U+0027
			buf.WriteString(`\'`)
		case '\\': // \ backslash (reverse solidus) U+005C
			buf.WriteString(`\\`)
		case '\x00', '\x01', '\x02', '\x03', '\x04', '\x05', '\x06', '\x07', '\x0b', '\x0e', '\x0f':
			// "00"-"07", "0b", "0e"-"0f"
			fmt.Fprintf(buf, `\u000%x`, r)
		default:
			buf.WriteRune(r)
		}
	}
	buf.WriteString("']")
}

// writePointerTo writes n to buf formatted as a [JSON Pointer] reference
// token. Defined by [NormalSelector].
//
// [JSON Pointer]: https://www.rfc-editor.org/rfc/rfc6901
func (n Name) writePointerTo(buf *strings.Builder) {
	buf.WriteString(strings.ReplaceAll(
		strings.ReplaceAll(string(n), "~", "~0"),
		"/", "~1",
	))
}

// WildcardSelector is a wildcard selector, e.g., * or [*], as defined by [RFC
// 9535 Section 2.3.2]. Interfaces implemented:
//   - [Selector]
//   - [fmt.Stringer]
//
// [RFC 9535 Section 2.3.2]: https://www.rfc-editor.org/rfc/rfc9535.html#name-wildcard-selector
type WildcardSelector struct{}

//nolint:gochecknoglobals
var wc = WildcardSelector{}

// Wildcard returns a [WildcardSelector] singleton.
func Wildcard() WildcardSelector { return wc }

// writeTo writes "*" to buf. Defined by [stringWriter].
func (WildcardSelector) writeTo(buf *strings.Builder) { buf.WriteByte('*') }

// String returns "*".
func (WildcardSelector) String() string { return "*" }

// isSingular returns false because a wild card can select more than one value
// from an object or array. Defined by the [Selector] interface.
func (WildcardSelector) isSingular() bool { return false }

// Select selects the values from input and returns them in a slice. Returns
// an empty slice if input is not []any map[string]any. Defined by the
// [Selector] interface.
func (WildcardSelector) Select(input, _ any) []any {
	switch val := input.(type) {
	case []any:
		return val
	case map[string]any:
		vals := make([]any, 0, len(val))
		for _, v := range val {
			vals = append(vals, v)
		}
		return vals
	}
	return make([]any, 0)
}

// SelectLocated selects the values from input and returns them with their
// normalized paths in a slice of [LocatedNode] values. Returns an empty
// slice if input is not []any map[string]any. Defined by the [Selector]
// interface.
func (WildcardSelector) SelectLocated(input, _ any, parent NormalizedPath) []*LocatedNode {
	switch val := input.(type) {
	case []any:
		vals := make([]*LocatedNode, len(val))
		for i, v := range val {
			vals[i] = newLocatedNode(append(parent, Index(i)), v)
		}
		return vals
	case map[string]any:
		vals := make([]*LocatedNode, 0, len(val))
		for k, v := range val {
			vals = append(vals, newLocatedNode(append(parent, Name(k)), v))
		}
		return vals
	}
	return make([]*LocatedNode, 0)
}

// Index is an array index selector, e.g., [3], as defined by [RFC
// 9535 Section 2.3.3]. Interfaces
// implemented:
//   - [Selector]
//   - [fmt.Stringer]
//   - [NormalSelector]
//
// [RFC 9535 Section 2.3.3]: https://www.rfc-editor.org/rfc/rfc9535.html#name-index-selector
type Index int

// isSingular returns true because Index selects a single value from an array.
// Defined by the [Selector] interface.
func (Index) isSingular() bool { return true }

// writeTo writes a string representation of i to buf. Defined by
// [stringWriter].
func (i Index) writeTo(buf *strings.Builder) {
	buf.WriteString(i.String())
}

// String returns a string representation of i.
func (i Index) String() string { return strconv.FormatInt(int64(i), 10) }

// Select selects i from input and returns it as a single value in a slice.
// Returns an empty slice if input is not a slice or if i it outside the
// bounds of input. Defined by the [Selector] interface.
func (i Index) Select(input, _ any) []any {
	if val, ok := input.([]any); ok {
		idx := int(i)
		if idx < 0 {
			if idx = len(val) + idx; idx >= 0 {
				return []any{val[idx]}
			}
		} else if idx < len(val) {
			return []any{val[idx]}
		}
	}
	return make([]any, 0)
}

// SelectLocated selects i from input and returns it with its normalized path
// as a single [LocatedNode] in a slice. Returns an empty slice if input is
// not a slice or if i it outside the bounds of input. Defined by the
// [Selector] interface.
func (i Index) SelectLocated(input, _ any, parent NormalizedPath) []*LocatedNode {
	if val, ok := input.([]any); ok {
		idx := int(i)
		if idx < 0 {
			if idx = len(val) + idx; idx >= 0 {
				return []*LocatedNode{newLocatedNode(append(parent, Index(idx)), val[idx])}
			}
		} else if idx < len(val) {
			return []*LocatedNode{newLocatedNode(append(parent, Index(idx)), val[idx])}
		}
	}
	return make([]*LocatedNode, 0)
}

// writeNormalizedTo writes n to buf formatted as a [normalized path] element.
// Implements [NormalSelector].
//
// [normalized path]: https://www.rfc-editor.org/rfc/rfc9535#section-2.7
func (i Index) writeNormalizedTo(buf *strings.Builder) {
	buf.WriteRune('[')
	buf.WriteString(strconv.FormatInt(int64(i), 10))
	buf.WriteRune(']')
}

// writePointerTo writes n to buf formatted as a [JSON Pointer] reference
// token. Defined by [NormalSelector].
//
// [JSON Pointer]: https://www.rfc-editor.org/rfc/rfc6901
func (i Index) writePointerTo(buf *strings.Builder) {
	buf.WriteString(strconv.FormatInt(int64(i), 10))
}

// SliceSelector is a slice selector, e.g., [0:100:5], as defined by [RFC
// 9535 Section 2.3.4]. Interfaces implemented:
//   - [Selector]
//   - [fmt.Stringer]
//
// [RFC 9535 Section 2.3.4]: https://www.rfc-editor.org/rfc/rfc9535.html#name-array-slice-selector
type SliceSelector struct {
	// Start of the slice; defaults to 0.
	start int
	// End of the slice; defaults to math.MaxInt.
	end int
	// Steps between start and end; defaults to 0.
	step int
}

// isSingular returns false because a slice selector can select more than one
// value from an array. Defined by the [Selector] interface.
func (SliceSelector) isSingular() bool { return false }

// Slice creates a new [SliceSelector]. Pass up to three integers or nils for
// the start, end, and step arguments. Subsequent arguments are ignored.
func Slice(args ...any) SliceSelector {
	const (
		startArg = 0
		endArg   = 1
		stepArg  = 2
	)
	// Set defaults.
	s := SliceSelector{0, math.MaxInt, 1}
	switch len(args) - 1 {
	case stepArg:
		//nolint:gosec // disable G602 https://github.com/securego/gosec/issues/1250
		switch step := args[stepArg].(type) {
		case int:
			s.step = step
		case nil:
			// Nothing to do
		default:
			panic("Third value passed to Slice is not an integer")
		}
		fallthrough
	case endArg:
		//nolint:gosec // disable G602 https://github.com/securego/gosec/issues/1250
		switch end := args[endArg].(type) {
		case int:
			s.end = end
		case nil:
			// Negative step: end with minimum int.
			if s.step < 0 {
				s.end = math.MinInt
			}
		default:
			panic("Second value passed to Slice is not an integer")
		}
		fallthrough
	case startArg:
		switch start := args[startArg].(type) {
		case int:
			s.start = start
		case nil:
			// Negative step: start with maximum int.
			if s.step < 0 {
				s.start = math.MaxInt
			}
		default:
			panic("First value passed to Slice is not an integer")
		}
	}
	return s
}

// writeTo writes a string representation of s to buf. Defined by
// [stringWriter].
func (s SliceSelector) writeTo(buf *strings.Builder) {
	if s.start != 0 && (s.step >= 0 || s.start != math.MaxInt) {
		buf.WriteString(strconv.FormatInt(int64(s.start), 10))
	}
	buf.WriteByte(':')
	if s.end != math.MaxInt && (s.step >= 0 || s.end != math.MinInt) {
		buf.WriteString(strconv.FormatInt(int64(s.end), 10))
	}
	if s.step != 1 {
		buf.WriteByte(':')
		buf.WriteString(strconv.FormatInt(int64(s.step), 10))
	}
}

// String returns a quoted string representation of s.
func (s SliceSelector) String() string {
	buf := new(strings.Builder)
	s.writeTo(buf)
	return buf.String()
}

// Select selects and returns the values from input for the indexes specified
// by s. Returns an empty slice if input is not a slice. Indexes outside the
// bounds of input will not be included in the return value. Defined by the
// [Selector] interface.
func (s SliceSelector) Select(input, _ any) []any {
	if val, ok := input.([]any); ok {
		lower, upper := s.Bounds(len(val))
		res := make([]any, 0, len(val))
		switch {
		case s.step > 0:
			for i := lower; i < upper; i += s.step {
				res = append(res, val[i])
			}
		case s.step < 0:
			for i := upper; lower < i; i += s.step {
				res = append(res, val[i])
			}
		}
		return res
	}
	return make([]any, 0)
}

// SelectLocated selects values from input for the indexes specified by s and
// returns thm with their normalized paths as [LocatedNode] values. Returns
// an empty slice if input is not a slice. Indexes outside the bounds of input
// will not be included in the return value. Defined by the [Selector]
// interface.
func (s SliceSelector) SelectLocated(input, _ any, parent NormalizedPath) []*LocatedNode {
	if val, ok := input.([]any); ok {
		lower, upper := s.Bounds(len(val))
		res := make([]*LocatedNode, 0, len(val))
		switch {
		case s.step > 0:
			for i := lower; i < upper; i += s.step {
				res = append(res, newLocatedNode(append(parent, Index(i)), val[i]))
			}
		case s.step < 0:
			for i := upper; lower < i; i += s.step {
				res = append(res, newLocatedNode(append(parent, Index(i)), val[i]))
			}
		}
		return res
	}
	return make([]*LocatedNode, 0)
}

// Start returns the start position.
func (s SliceSelector) Start() int {
	return s.start
}

// End returns the end position.
func (s SliceSelector) End() int {
	return s.end
}

// Step returns the step value.
func (s SliceSelector) Step() int {
	return s.step
}

// Bounds returns the lower and upper bounds for selecting from a slice of
// length.
func (s SliceSelector) Bounds(length int) (int, int) {
	start := normalize(s.start, length)
	end := normalize(s.end, length)
	switch {
	case s.step > 0:
		return max(min(start, length), 0), max(min(end, length), 0)
	case s.step < 0:
		return max(min(end, length-1), -1), max(min(start, length-1), -1)
	default:
		return 0, 0
	}
}

// normalize normalizes index i relative to a slice of length.
func normalize(i, length int) int {
	if i >= 0 {
		return i
	}

	return length + i
}

// FilterSelector is a filter selector, e.g., ?(), as defined by [RFC
// 9535 Section 2.3.5]. Interfaces implemented:
//   - [Selector]
//   - [fmt.Stringer]
//
// [RFC 9535 Section 2.3.5]: https://www.rfc-editor.org/rfc/rfc9535.html#name-filter-selector
type FilterSelector struct {
	LogicalOr
}

// Filter returns a new [FilterSelector] that ORs the evaluation of each expr.
func Filter(expr ...LogicalAnd) *FilterSelector {
	return &FilterSelector{LogicalOr: expr}
}

// String returns a string representation of f.
func (f *FilterSelector) String() string {
	buf := new(strings.Builder)
	f.writeTo(buf)
	return buf.String()
}

// writeTo writes a string representation of f to buf. Defined by
// [stringWriter].
func (f *FilterSelector) writeTo(buf *strings.Builder) {
	buf.WriteRune('?')
	f.LogicalOr.writeTo(buf)
}

// Select selects and returns values that f filters from current. Filter
// expressions may evaluate the current value (@), the root value ($), or any
// path expression. Defined by the [Selector] interface.
func (f *FilterSelector) Select(current, root any) []any {
	ret := []any{}
	switch current := current.(type) {
	case []any:
		for _, v := range current {
			if f.Eval(v, root) {
				ret = append(ret, v)
			}
		}
	case map[string]any:
		for _, v := range current {
			if f.Eval(v, root) {
				ret = append(ret, v)
			}
		}
	}

	return ret
}

// SelectLocated selects and returns [LocatedNode] values with values that f
// filters from current. Filter expressions may evaluate the current value
// (@), the root value ($), or any path expression. Defined by the [Selector]
// interface.
func (f *FilterSelector) SelectLocated(current, root any, parent NormalizedPath) []*LocatedNode {
	ret := []*LocatedNode{}
	switch current := current.(type) {
	case []any:
		for i, v := range current {
			if f.Eval(v, root) {
				ret = append(ret, newLocatedNode(append(parent, Index(i)), v))
			}
		}
	case map[string]any:
		for k, v := range current {
			if f.Eval(v, root) {
				ret = append(ret, newLocatedNode(append(parent, Name(k)), v))
			}
		}
	}

	return ret
}

// Eval evaluates the f's [LogicalOr] expression against node and root. Uses
// [FilterSelector.Select] as it iterates over nodes, and always passes the
// root value($) for filter expressions that reference it.
func (f *FilterSelector) Eval(node, root any) bool {
	return f.testFilter(node, root)
}

// isSingular returns false because Filters can return more than one value.
// Defined by the [Selector] interface.
func (f *FilterSelector) isSingular() bool { return false }
