package util

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type csvColumn struct {
	Header     string
	FieldIndex int
}

// MarshalCSV marshals a struct slice/array to UTF-8 CSV bytes.
// It reads CSV headers from struct tag `csv:"header"`.
func MarshalCSV(rows interface{}) ([]byte, error) {
	value := reflect.ValueOf(rows)
	if !value.IsValid() {
		return nil, errors.New("rows is nil")
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return nil, errors.New("rows is nil")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return nil, errors.New("rows must be a slice or array")
	}

	elemType := value.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return nil, errors.New("rows element must be struct")
	}

	columns := extractCSVColumns(elemType)
	if len(columns) == 0 {
		return nil, errors.New("no csv tags found")
	}

	var buffer bytes.Buffer
	// UTF-8 BOM keeps Chinese headers readable in Excel on Windows.
	buffer.Write([]byte{0xEF, 0xBB, 0xBF})
	writer := csv.NewWriter(&buffer)

	headers := make([]string, 0, len(columns))
	for _, col := range columns {
		headers = append(headers, col.Header)
	}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	for i := 0; i < value.Len(); i++ {
		row := value.Index(i)
		if row.Kind() == reflect.Ptr {
			if row.IsNil() {
				if err := writer.Write(make([]string, len(columns))); err != nil {
					return nil, err
				}
				continue
			}
			row = row.Elem()
		}

		cells := make([]string, len(columns))
		for idx, col := range columns {
			field := row.Field(col.FieldIndex)
			cells[idx] = sanitizeCSVCell(reflectValueToString(field))
		}
		if err := writer.Write(cells); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func extractCSVColumns(elemType reflect.Type) []csvColumn {
	columns := make([]csvColumn, 0, elemType.NumField())
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		header := strings.TrimSpace(field.Tag.Get("csv"))
		if header == "" || header == "-" {
			continue
		}
		columns = append(columns, csvColumn{
			Header:     header,
			FieldIndex: i,
		})
	}
	return columns
}

func reflectValueToString(value reflect.Value) string {
	if !value.IsValid() {
		return ""
	}

	switch value.Kind() {
	case reflect.Interface, reflect.Ptr:
		if value.IsNil() {
			return ""
		}
		return reflectValueToString(value.Elem())
	}

	if value.CanInterface() {
		if t, ok := value.Interface().(time.Time); ok {
			return FormatDateTimeOrEmpty(t)
		}
		if stringer, ok := value.Interface().(fmt.Stringer); ok {
			return stringer.String()
		}
	}

	switch value.Kind() {
	case reflect.String:
		return value.String()
	case reflect.Bool:
		return strconv.FormatBool(value.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(value.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(value.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(value.Float(), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(value.Float(), 'f', -1, 64)
	default:
		if value.CanInterface() {
			return fmt.Sprint(value.Interface())
		}
		return ""
	}
}

func sanitizeCSVCell(raw string) string {
	if raw == "" {
		return ""
	}
	trimmed := strings.TrimLeft(raw, " \t\r\n")
	if trimmed == "" {
		return raw
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + raw
	default:
		return raw
	}
}
