package http

import (
	"encoding/json"
	"fmt"
)

func JSON(data any) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data to JSON: %w", err)
	}
	return string(jsonData), nil
}

func FromJSON[T any](data []byte) (T, error) {
	var result T
	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, fmt.Errorf("failed to unmarshal JSON data: %w", err)
	}
	return result, nil
}
