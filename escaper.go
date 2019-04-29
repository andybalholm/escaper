// The escaper package implements automatic contextual HTML escaping. It is
// derived from the auto-escaping logic in the html/template package, but it
// operates at run time rather than at the time of template compilation.
package escaper

import "io"

// An Escaper wraps an io.Writer and provides automatic contextual escaping
// for HTML output.
type Escaper struct {
	w   io.Writer
	ctx context
}

// New returns a new Escaper that wraps w.
func New(w io.Writer) *Escaper {
	return &Escaper{
		w: w,
	}
}

// Literal writes a string of literal HTML.
func (e *Escaper) Literal(s string) error {
	i := 0
	for i < len(s) {
		var n int
		e.ctx, n = contextAfterText(e.ctx, s[i:])
		i += n
	}
	if e.ctx.err != nil {
		return e.ctx.err
	}

	_, err := io.WriteString(e.w, s)
	return err
}

// Value escapes v as appropriate for the current context, and writes the
// result.
func (e *Escaper) Value(v interface{}) error {
	if e.ctx.state == stateBeforeValue {
		// Automatically double-quote attribute values.
		e.Literal(`"`)
		defer e.Literal(`"`)
	}

	e.ctx = nudge(e.ctx)
	s := make([]func(...interface{}) string, 0, 3)
	switch e.ctx.state {
	case stateError:
		return e.ctx.err
	case stateURL, stateCSSDqStr, stateCSSSqStr, stateCSSDqURL, stateCSSSqURL, stateCSSURL:
		switch e.ctx.urlPart {
		case urlPartNone:
			s = append(s, urlFilter)
			fallthrough
		case urlPartPreQuery:
			switch e.ctx.state {
			case stateCSSDqStr, stateCSSSqStr:
				s = append(s, cssEscaper)
			default:
				s = append(s, urlNormalizer)
			}
		case urlPartQueryOrFrag:
			s = append(s, urlEscaper)
		case urlPartUnknown:
			e.ctx = context{
				state: stateError,
				err:   errorf(ErrAmbigContext, "tried to print %v in an ambiguous URL context", v),
			}
			return e.ctx.err
		default:
			panic(e.ctx.urlPart.String())
		}
	case stateJS:
		s = append(s, jsValEscaper)
		// A slash after a value starts a div operator.
		e.ctx.jsCtx = jsCtxDivOp
	case stateJSDqStr, stateJSSqStr:
		s = append(s, jsStrEscaper)
	case stateJSRegexp:
		s = append(s, jsRegexpEscaper)
	case stateCSS:
		s = append(s, cssValueFilter)
	case stateText:
		s = append(s, htmlEscaper)
	case stateRCDATA:
		s = append(s, rcdataEscaper)
	case stateAttr:
		// Handled below in delim check.
	case stateAttrName, stateTag:
		e.ctx.state = stateAttrName
		s = append(s, htmlNameFilter)
	default:
		if isComment(e.ctx.state) {
			s = append(s, commentEscaper)
		} else {
			panic("unexpected state " + e.ctx.state.String())
		}
	}
	switch e.ctx.delim {
	case delimNone:
		// No extra-escaping needed for raw text content.
	case delimSpaceOrTagEnd:
		s = append(s, htmlNospaceEscaper)
	default:
		s = append(s, attrEscaper)
	}

	for _, filter := range s {
		v = filter(v)
	}
	if len(s) == 0 {
		v, _ = stringify(v)
	}

	return e.Literal(v.(string))
}

// Print writes some HTML. It interprets its arguments as an alternating list
// of strings of literal HTML and values that need to be escaped.
func (e *Escaper) Print(args ...interface{}) error {
	prevWasLiteral := false
	for _, v := range args {
		switch v := v.(type) {
		case string:
			if prevWasLiteral {
				err := e.Value(v)
				if err != nil {
					return err
				}
				prevWasLiteral = false
			} else {
				err := e.Literal(v)
				if err != nil {
					return err
				}
				prevWasLiteral = true
			}

		case List:
			err := e.Print([]interface{}(v)...)
			if err != nil {
				return err
			}
			prevWasLiteral = false

		default:
			err := e.Value(v)
			if err != nil {
				return err
			}
			prevWasLiteral = false
		}
	}
	return nil
}

// A List is a prepared argument list for Escaper.Print. It can be nested
// within another call to Print.
type List []interface{}

// Write bypasses the escaper, and writes directly to the underlying Writer.
// This is useful if part of your page is rendered with templates, or some
// other library that expects a Writer.
func (e *Escaper) Write(p []byte) (n int, err error) {
	return e.w.Write(p)
}
