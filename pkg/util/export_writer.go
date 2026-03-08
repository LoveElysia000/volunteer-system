package util

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type exportColumn struct {
	Header     string
	FieldIndex int
}

// MarshalXLSX marshals a struct slice/array to XLSX bytes.
// It reads column headers from struct tag `excel:"header"`.
func MarshalXLSX(rows interface{}) ([]byte, error) {
	return marshalXLSX(rows, false)
}

// MarshalXLSXWithFrozenHeader marshals rows and freezes the first header row.
func MarshalXLSXWithFrozenHeader(rows interface{}) ([]byte, error) {
	return marshalXLSX(rows, true)
}

func marshalXLSX(rows interface{}, freezeHeader bool) ([]byte, error) {
	value, elemType, err := resolveExportRows(rows)
	if err != nil {
		return nil, err
	}

	columns := extractExcelColumns(elemType)
	if len(columns) == 0 {
		return nil, errors.New("no excel tags found")
	}

	file := excelize.NewFile()
	sheetName := file.GetSheetName(file.GetActiveSheetIndex())
	if freezeHeader {
		if err := file.SetPanes(sheetName, &excelize.Panes{
			Freeze:      true,
			YSplit:      1,
			TopLeftCell: "A2",
			ActivePane:  "bottomLeft",
		}); err != nil {
			return nil, err
		}
	}

	for colIdx, col := range columns {
		cellName, cellErr := excelize.CoordinatesToCellName(colIdx+1, 1)
		if cellErr != nil {
			return nil, cellErr
		}
		if err = file.SetCellStr(sheetName, cellName, col.Header); err != nil {
			return nil, err
		}
	}

	writeRow := 2
	for rowIdx := 0; rowIdx < value.Len(); rowIdx++ {
		row := value.Index(rowIdx)
		if row.Kind() == reflect.Ptr {
			if row.IsNil() {
				continue
			}
			row = row.Elem()
		}

		for colIdx, col := range columns {
			cellName, cellErr := excelize.CoordinatesToCellName(colIdx+1, writeRow)
			if cellErr != nil {
				return nil, cellErr
			}
			field := row.Field(col.FieldIndex)
			if err = file.SetCellValue(sheetName, cellName, reflectValueForExcel(field)); err != nil {
				return nil, err
			}
		}
		writeRow++
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func resolveExportRows(rows interface{}) (reflect.Value, reflect.Type, error) {
	value := reflect.ValueOf(rows)
	if !value.IsValid() {
		return reflect.Value{}, nil, errors.New("rows is nil")
	}
	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Value{}, nil, errors.New("rows is nil")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Slice && value.Kind() != reflect.Array {
		return reflect.Value{}, nil, errors.New("rows must be a slice or array")
	}

	elemType := value.Type().Elem()
	if elemType.Kind() == reflect.Ptr {
		elemType = elemType.Elem()
	}
	if elemType.Kind() != reflect.Struct {
		return reflect.Value{}, nil, errors.New("rows element must be struct")
	}

	return value, elemType, nil
}

func extractExcelColumns(elemType reflect.Type) []exportColumn {
	columns := make([]exportColumn, 0, elemType.NumField())
	for i := 0; i < elemType.NumField(); i++ {
		field := elemType.Field(i)
		if field.PkgPath != "" {
			continue
		}
		header := strings.TrimSpace(field.Tag.Get("excel"))
		if header == "" || header == "-" {
			continue
		}
		columns = append(columns, exportColumn{
			Header:     header,
			FieldIndex: i,
		})
	}
	return columns
}

func reflectValueForExcel(value reflect.Value) interface{} {
	if !value.IsValid() {
		return ""
	}

	switch value.Kind() {
	case reflect.Interface, reflect.Ptr:
		if value.IsNil() {
			return ""
		}
		return reflectValueForExcel(value.Elem())
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
		return value.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return value.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return value.Uint()
	case reflect.Float32:
		return value.Float()
	case reflect.Float64:
		return value.Float()
	default:
		if value.CanInterface() {
			return fmt.Sprint(value.Interface())
		}
		return ""
	}
}
