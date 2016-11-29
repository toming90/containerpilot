package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mitchellh/mapstructure"
)

func StrToCmd(command string) *exec.Cmd {
	if command != "" {
		return ArgsToCmd(strings.Split(strings.TrimSpace(command), " "))
	}
	return nil
}

func ParseCommandArgs(raw json.RawMessage) (*exec.Cmd, error) {
	if raw == nil {
		return nil, nil
	}
	// Parse as a string
	var stringCmd string
	if err := json.Unmarshal(raw, &stringCmd); err == nil {
		return StrToCmd(stringCmd), nil
	}

	var arrayCmd []string
	if err := json.Unmarshal(raw, &arrayCmd); err == nil {
		return ArgsToCmd(arrayCmd), nil
	}
	return nil, errors.New("Command argument must be a string or an array")
}

func ArgsToCmd(args []string) *exec.Cmd {
	if len(args) == 0 {
		return nil
	}
	if len(args) > 1 {
		return exec.Command(args[0], args[1:]...)
	}
	return exec.Command(args[0])
}

// DecodeRaw decodes a raw interface into the target structure
func DecodeRaw(raw interface{}, result interface{}) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		ErrorUnused:      true,
		WeaklyTypedInput: true,
		Result:           result,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(raw)
}

// ToStringArray converts the given interface to a []string if possible
func ToStringArray(raw interface{}) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	switch t := raw.(type) {
	case string:
		return []string{t}, nil
	case []string:
		return t, nil
	case []interface{}:
		return interfaceToStringArray(t), nil
	default:
		return nil, fmt.Errorf("Unexpected argument type: %T", t)
	}
}

func interfaceToString(raw interface{}) string {
	switch t := raw.(type) {
	case string:
		return t
	default:
		return fmt.Sprintf("%v", t)
	}
}

func interfaceToStringArray(rawArray []interface{}) []string {
	if rawArray == nil || len(rawArray) == 0 {
		return nil
	}
	var stringArray []string
	for _, raw := range rawArray {
		stringArray = append(stringArray, interfaceToString(raw))
	}
	return stringArray
}
