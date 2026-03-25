package validator

import (
	"encoding/json"
	"fmt"
	"go_project/config"
)

type Violation struct {
	Field    string
	Issue    string
	Expected string
	Got      string
}

func ValidateJSON(body []byte, schema map[string]string, direction string, contract *config.Contract) []Violation {
	var violations []Violation

	// Parse the incoming JSON into a map
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		violations = append(violations, Violation{
			Field:    "body",
			Issue:    "invalid JSON",
			Expected: "valid JSON",
			Got:      string(body),
		})
		return violations
	}

	// Check every field defined in the contract
	for field, expectedType := range schema {

		val, exists := data[field]

		// Check if field is missing
		if !exists {
			violations = append(violations, Violation{
				Field:    field,
				Issue:    "missing field",
				Expected: expectedType,
				Got:      "null",
			})
			continue
		}

		// Check if field type matches
		actualType := getType(val)
		if actualType != expectedType {
			violations = append(violations, Violation{
				Field:    field,
				Issue:    "wrong type",
				Expected: expectedType,
				Got:      actualType,
			})
		}
	}

	// Print results
	if len(violations) == 0 {
		fmt.Printf("[%s] Contract OK\n", direction)
	} else {
		fmt.Printf("[%s] Contract VIOLATIONS FOUND:\n", direction)
		for _, v := range violations {
			fmt.Printf("  - Field: %s | Issue: %s | Expected: %s | Got: %s\n",
				v.Field, v.Issue, v.Expected, v.Got)
		}
	}

	return violations
}

// getType returns a simple type string for a JSON value
func getType(val interface{}) string {
	switch val.(type) {
	case string:
		return "string"
	case float64:
		return "number" // JSON numbers are float64 in Go
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
