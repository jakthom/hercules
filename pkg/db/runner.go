package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
)

type Table struct {
	Columns []string
	Rows    [][]any
}

type Executor interface {
	Exec(context.Context, SQL) error
	Query(context.Context, SQL, int) (Table, error)
}

const maxExactlyRepresentableInteger = int64(1 << 53)

func QueryTable(ctx context.Context, conn *sql.Conn, query SQL, maxRows int) (Table, error) {
	rows, err := conn.QueryContext(ctx, string(query))
	if err != nil {
		return Table{}, err
	}
	table, readErr := readTable(rows, maxRows)
	closeErr := rows.Close()
	return table, errors.Join(readErr, closeErr)
}

func readTable(rows *sql.Rows, maxRows int) (Table, error) {
	columns, err := rows.Columns()
	if err != nil {
		return Table{}, err
	}
	table := Table{Columns: columns}
	for rows.Next() {
		if len(table.Rows) >= maxRows {
			return Table{}, fmt.Errorf("query exceeded maximum of %d rows", maxRows)
		}
		values := make([]any, len(columns))
		destinations := make([]any, len(columns))
		for i := range values {
			destinations[i] = &values[i]
		}
		if err := rows.Scan(destinations...); err != nil {
			return Table{}, err
		}
		table.Rows = append(table.Rows, values)
	}
	if err := rows.Err(); err != nil {
		return Table{}, err
	}
	return table, nil
}

func ToFloat64(value any) (float64, error) {
	if value == nil {
		return 0, errors.New("metric value is null")
	}
	var result float64
	var conversionErr error
	switch typed := value.(type) {
	case float64:
		result = typed
	case float32:
		result = float64(typed)
	case int:
		result, conversionErr = signedIntegerToFloat64(int64(typed))
	case int8:
		result = float64(typed)
	case int16:
		result = float64(typed)
	case int32:
		result = float64(typed)
	case int64:
		result, conversionErr = signedIntegerToFloat64(typed)
	case uint:
		result, conversionErr = unsignedIntegerToFloat64(uint64(typed))
	case uint8:
		result = float64(typed)
	case uint16:
		result = float64(typed)
	case uint32:
		result = float64(typed)
	case uint64:
		result, conversionErr = unsignedIntegerToFloat64(typed)
	case *big.Int:
		converted, accuracy := new(big.Float).SetInt(typed).Float64()
		if accuracy != big.Exact {
			return 0, fmt.Errorf("integer %s cannot be represented exactly as float64", typed.String())
		}
		result = converted
	case *big.Float:
		converted, accuracy := typed.Float64()
		if accuracy != big.Exact {
			return 0, fmt.Errorf("number %s cannot be represented exactly as float64", typed.String())
		}
		result = converted
	case string:
		converted, err := strconv.ParseFloat(typed, 64)
		if err != nil {
			return 0, fmt.Errorf("parse metric value %q: %w", typed, err)
		}
		result = converted
	default:
		return 0, fmt.Errorf("unsupported metric value type %T", value)
	}
	if conversionErr != nil {
		return 0, conversionErr
	}
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, fmt.Errorf("metric value must be finite, got %v", result)
	}
	return result, nil
}

func signedIntegerToFloat64(value int64) (float64, error) {
	if value < -maxExactlyRepresentableInteger || value > maxExactlyRepresentableInteger {
		return 0, fmt.Errorf("integer %d exceeds the exact float64 integer range", value)
	}
	return float64(value), nil
}

func unsignedIntegerToFloat64(value uint64) (float64, error) {
	if value > uint64(maxExactlyRepresentableInteger) {
		return 0, fmt.Errorf("integer %d exceeds the exact float64 integer range", value)
	}
	return float64(value), nil
}

func ToLabelValue(value any) (string, error) {
	if value == nil {
		return "NULL", nil
	}
	switch typed := value.(type) {
	case string:
		return typed, nil
	case []byte:
		return string(typed), nil
	case fmt.Stringer:
		return typed.String(), nil
	case bool:
		return strconv.FormatBool(typed), nil
	case int64:
		return strconv.FormatInt(typed, 10), nil
	case int8:
		return strconv.FormatInt(int64(typed), 10), nil
	case int16:
		return strconv.FormatInt(int64(typed), 10), nil
	case int32:
		return strconv.FormatInt(int64(typed), 10), nil
	case int:
		return strconv.Itoa(typed), nil
	case uint64:
		return strconv.FormatUint(typed, 10), nil
	case uint8:
		return strconv.FormatUint(uint64(typed), 10), nil
	case uint16:
		return strconv.FormatUint(uint64(typed), 10), nil
	case uint32:
		return strconv.FormatUint(uint64(typed), 10), nil
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64), nil
	case float32:
		return strconv.FormatFloat(float64(typed), 'g', -1, 32), nil
	default:
		return "", fmt.Errorf("unsupported label value type %T", value)
	}
}
