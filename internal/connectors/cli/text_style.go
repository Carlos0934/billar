package cli

import (
	"io"
	"os"
	"strings"
)

type textView struct {
	style ansiStyle
	buf   strings.Builder
}

func newTextView(out io.Writer, colorEnabled bool) *textView {
	return &textView{style: newANSIStyle(out, colorEnabled)}
}

func (v *textView) Title(value string) *textView {
	v.writeLine(v.style.Title(value))
	return v
}

func (v *textView) Divider(value string) *textView {
	v.writeLine(v.style.Muted(value))
	return v
}

func (v *textView) Field(label, value string) *textView {
	v.writeLine(v.style.Label(label) + ": " + value)
	return v
}

func (v *textView) Status(label, value string) *textView {
	v.writeLine(v.style.Label(label) + ": " + v.style.Status(value))
	return v
}

func (v *textView) Line(value string) *textView {
	v.writeLine(value)
	return v
}

func (v *textView) Build() string {
	return v.buf.String()
}

func (v *textView) writeLine(value string) {
	v.buf.WriteString(value)
	v.buf.WriteByte('\n')
}

type ansiStyle struct {
	enabled bool
}

func newANSIStyle(out io.Writer, colorEnabled bool) ansiStyle {
	if !colorEnabled {
		return ansiStyle{}
	}

	if _, ok := out.(*os.File); !ok {
		return ansiStyle{}
	}

	return ansiStyle{enabled: true}
}

func (s ansiStyle) Title(value string) string {
	return s.wrap("1;36", value)
}

func (s ansiStyle) Muted(value string) string {
	return s.wrap("2", value)
}

func (s ansiStyle) Label(value string) string {
	return s.wrap("1", value)
}

func (s ansiStyle) Status(value string) string {
	if strings.EqualFold(strings.TrimSpace(value), "ok") {
		return s.wrap("32", value)
	}

	return s.wrap("33", value)
}

func (s ansiStyle) wrap(code, value string) string {
	if !s.enabled {
		return value
	}

	return "\x1b[" + code + "m" + value + "\x1b[0m"
}
