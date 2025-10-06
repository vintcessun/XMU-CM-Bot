package utils

import "encoding/json"

// UnmarshalJSON 泛型函数，将JSON数据反序列化为指定类型
func UnmarshalJSON[T any](data []byte) (*T, error) {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return &result, err
	}
	return &result, nil
}

func MarshalJSON[T any](data any) (string, error) {
	jsonBytes, err := MarshalJSONByte[T](data)
	return string(jsonBytes), err
}

func MarshalJSONByte[T any](data any) ([]byte, error) {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}
