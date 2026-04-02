package cli

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strconv"
	"strings"

	toon "github.com/toon-format/toon-go"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
	FormatTOON Format = "toon"
)

type Formatter interface {
	Write(io.Writer, OutputResult) error
}

type TextWriter func(io.Writer) error

type OutputResult struct {
	Payload    any
	TextWriter TextWriter
}

func (r OutputResult) CanonicalPayload() any {
	return r.Payload
}

func ParseFormat(value string) (Format, error) {
	switch Format(strings.ToLower(strings.TrimSpace(value))) {
	case "", FormatText:
		return FormatText, nil
	case FormatJSON:
		return FormatJSON, nil
	case FormatTOON:
		return FormatTOON, nil
	default:
		return "", fmt.Errorf("unsupported output format %q (want text, json, or toon)", value)
	}
}

func NewFormatter(format Format) (Formatter, error) {
	switch format {
	case FormatText:
		return textFormatter{}, nil
	case FormatJSON:
		return jsonFormatter{}, nil
	case FormatTOON:
		return toonFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported output format %q", format)
	}
}

func WriteOutput(out io.Writer, format Format, result OutputResult) error {
	formatter, err := NewFormatter(format)
	if err != nil {
		return err
	}

	return formatter.Write(out, result)
}

type textFormatter struct{}

func (textFormatter) Write(out io.Writer, result OutputResult) error {
	if result.TextWriter != nil {
		return result.TextWriter(out)
	}

	return writeGenericText(out, result.CanonicalPayload())
}

type jsonFormatter struct{}

func (jsonFormatter) Write(out io.Writer, result OutputResult) error {
	return json.NewEncoder(out).Encode(result.CanonicalPayload())
}

type toonFormatter struct{}

func (toonFormatter) Write(out io.Writer, result OutputResult) error {
	encoded, err := toon.Marshal(result.CanonicalPayload())
	if err != nil {
		return fmt.Errorf("marshal toon output: %w", err)
	}

	if _, err := out.Write(encoded); err != nil {
		return err
	}

	if len(encoded) == 0 || encoded[len(encoded)-1] != '\n' {
		if _, err := out.Write([]byte{'\n'}); err != nil {
			return err
		}
	}

	return nil
}

func writeGenericText(out io.Writer, payload any) error {
	value := indirectValue(reflect.ValueOf(payload))
	if !value.IsValid() {
		return errors.New("text output requires a canonical payload")
	}

	if rendered, ok, err := renderTextScalar(value); ok || err != nil {
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(out, rendered)
		return err
	}

	switch value.Kind() {
	case reflect.Struct:
		return writeGenericStruct(out, value)
	case reflect.Map:
		return writeGenericMap(out, value)
	default:
		return fmt.Errorf("text output fallback does not support %s", value.Type())
	}
}

func writeGenericStruct(out io.Writer, value reflect.Value) error {
	valueType := value.Type()
	for i := range value.NumField() {
		fieldType := valueType.Field(i)
		if !fieldType.IsExported() {
			continue
		}

		name, ok := fallbackFieldName(fieldType)
		if !ok {
			continue
		}

		rendered, ok, err := renderTextScalar(value.Field(i))
		if err != nil {
			return fmt.Errorf("render text field %q: %w", name, err)
		}
		if !ok {
			return fmt.Errorf("text output fallback does not support field %q of type %s", name, indirectType(value.Field(i).Type()))
		}

		if _, err := fmt.Fprintf(out, "%s: %s\n", name, rendered); err != nil {
			return err
		}
	}

	return nil
}

func writeGenericMap(out io.Writer, value reflect.Value) error {
	if value.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("text output fallback does not support %s", value.Type())
	}

	keys := value.MapKeys()
	sortedKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		sortedKeys = append(sortedKeys, key.String())
	}
	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		rendered, ok, err := renderTextScalar(value.MapIndex(reflect.ValueOf(key)))
		if err != nil {
			return fmt.Errorf("render text field %q: %w", key, err)
		}
		if !ok {
			return fmt.Errorf("text output fallback does not support field %q of type %s", key, indirectType(value.MapIndex(reflect.ValueOf(key)).Type()))
		}

		if _, err := fmt.Fprintf(out, "%s: %s\n", key, rendered); err != nil {
			return err
		}
	}

	return nil
}

func renderTextScalar(value reflect.Value) (string, bool, error) {
	value = indirectValue(value)
	if !value.IsValid() {
		return "<nil>", true, nil
	}

	if value.CanInterface() {
		if textMarshaler, ok := value.Interface().(encoding.TextMarshaler); ok {
			text, err := textMarshaler.MarshalText()
			if err != nil {
				return "", false, err
			}
			return string(text), true, nil
		}

		if stringer, ok := value.Interface().(fmt.Stringer); ok {
			return stringer.String(), true, nil
		}
	}

	switch value.Kind() {
	case reflect.String:
		return value.String(), true, nil
	case reflect.Bool:
		return strconv.FormatBool(value.Bool()), true, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10), true, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(value.Uint(), 10), true, nil
	case reflect.Float32:
		return strconv.FormatFloat(value.Float(), 'f', -1, 32), true, nil
	case reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64), true, nil
	default:
		return "", false, nil
	}
}

func fallbackFieldName(field reflect.StructField) (string, bool) {
	for _, tagName := range []string{"json", "toon"} {
		tag := field.Tag.Get(tagName)
		if tag == "-" {
			return "", false
		}

		name := strings.TrimSpace(strings.Split(tag, ",")[0])
		if name != "" {
			return name, true
		}
	}

	return strings.ToLower(field.Name), true
}

func indirectValue(value reflect.Value) reflect.Value {
	for value.IsValid() && (value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer) {
		if value.IsNil() {
			return reflect.Value{}
		}
		value = value.Elem()
	}

	return value
}

func indirectType(valueType reflect.Type) reflect.Type {
	for valueType.Kind() == reflect.Pointer {
		valueType = valueType.Elem()
	}

	return valueType
}
