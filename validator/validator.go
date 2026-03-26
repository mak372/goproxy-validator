package validator

import (
	"encoding/json"
	"go_project/config"
	"go_project/logger"
)

type Violation struct {
	Field    string
	Issue    string
	Expected string
	Got      string
}

func ValidateJSON(body []byte, schema map[string]string, direction string, contract *config.Contract) []Violation {
	var violations []Violation

	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		violations = append(violations, Violation{
			Field:    "body",
			Issue:    "invalid JSON",
			Expected: "valid JSON",
			Got:      string(body),
		})
		logger.LogViolation(contract.Endpoint, contract.Method, direction, "body", "invalid JSON", "valid JSON", string(body))
		return violations
	}

	// Check every field in contract
	for field, expectedType := range schema {
		val, exists := data[field]

		if !exists {
			violations = append(violations, Violation{
				Field:    field,
				Issue:    "missing field",
				Expected: expectedType,
				Got:      "null",
			})
			logger.LogViolation(contract.Endpoint, contract.Method, direction, field, "missing field", expectedType, "null")
			continue
		}

		actualType := getType(val)
		if actualType != expectedType {
			violations = append(violations, Violation{
				Field:    field,
				Issue:    "wrong type",
				Expected: expectedType,
				Got:      actualType,
			})
			logger.LogViolation(contract.Endpoint, contract.Method, direction, field, "wrong type", expectedType, actualType)
		}
	}

	if len(violations) == 0 {
		logger.LogOK(contract.Endpoint, contract.Method, direction)
	}

	return violations
}

func getType(val interface{}) string {
	switch val.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case map[string]interface{}:
		return "object"
	case []interface{}:
		return "array"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}
