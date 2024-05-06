package k8sresmetric

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func ConvertUnit(unit string, val interface{}) (float64, error) {
	var convertedVal float64
	value, err := CoerceToFloat64(val)
	if err != nil {
		return 0.0, err
	}

	switch unit {
	case "ms":
		convertedVal = msToSec(value)
	case "Mb":
		convertedVal = mbToBytes(value)
	case "MiB":
		convertedVal = mibToBytes(value)
	case "microsec":
		convertedVal = microsecToSec(value)
	case "bytes", "count":
		convertedVal = value
	default:
		convertedVal = value
	}
	return convertedVal, nil
}
func msToSec(value float64) float64 {
	return value / 1000
}

// microsecToSec converts millisecond to seconds
func microsecToSec(value float64) float64 {
	return value / 1000000
}

// mbToBytes converts Mb to Bytes
func mbToBytes(value float64) float64 {
	return value * 1000000
}

// mibToBytes converts Mib to Bytes
func mibToBytes(value float64) float64 {
	return value * 1048576
}

func CoerceToFloat64(val interface{}) (float64, error) {

	switch t := val.(type) {
	case float32:
		return float64(t), nil
	case float64:
		return t, nil
	case int:
		return float64(t), nil
	case int8:
		return float64(t), nil
	case int32:
		return float64(t), nil
	case int64:
		return float64(t), nil
	case uint:
		return float64(t), nil
	case uint8:
		return float64(t), nil
	case uint16:
		return float64(t), nil
	case uint32:
		return float64(t), nil
	case uint64:
		return float64(t), nil
	case json.Number:
		return t.Float64()
	case string:
		valStr := strings.ToLower(t)

		if valStr == "true" || valStr == "ready" {
			valStr = "1.0"
		} else if valStr == "false" || valStr == "notready" {
			valStr = "0.0"
		}
		return strconv.ParseFloat(valStr, 64)
	case bool:
		if t {
			return 1.0, nil
		}
		return 0.0, nil
	case nil:
		return 0.0, nil
	default:
		return 0.0, fmt.Errorf("unable to coerce %#v to float64", val)
	}
}
