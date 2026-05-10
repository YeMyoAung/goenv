package goenv

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/joho/godotenv"
)

type Args struct {
	FileName string
	Validate *validator.Validate
}

func loadFromEnv() map[string]string {
	envMap := make(map[string]string)

	for _, v := range os.Environ() {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	return envMap
}

func decodeEnvToStruct[T any](env map[string]string) (*T, error) {
	var config T

	value := reflect.ValueOf(&config).Elem()
	if value.Kind() != reflect.Struct {
		return nil, errors.New("config must be a struct")
	}

	if err := fillStruct(value, env); err != nil {
		return nil, err
	}

	return &config, nil
}

func fillStruct(value reflect.Value, env map[string]string) error {
	valueType := value.Type()

	for i := 0; i < value.NumField(); i++ {
		field := value.Field(i)
		structField := valueType.Field(i)

		if !field.CanSet() {
			continue
		}

		// Handle embedded pointer structs:
		//
		// type Config struct {
		//     *GlobalDBConfig
		//     *RedisConfig
		// }
		if field.Kind() == reflect.Ptr &&
			field.Type().Elem().Kind() == reflect.Struct &&
			field.Type().Elem() != reflect.TypeOf(time.Duration(0)) {

			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}

			if err := fillStruct(field.Elem(), env); err != nil {
				return err
			}

			continue
		}

		// Handle normal nested structs:
		//
		// type Config struct {
		//     DB GlobalDBConfig
		// }
		if field.Kind() == reflect.Struct &&
			field.Type() != reflect.TypeOf(time.Duration(0)) {

			if err := fillStruct(field, env); err != nil {
				return err
			}

			continue
		}

		envKey := getEnvKey(structField)
		if envKey == "" {
			continue
		}

		rawValue, ok := env[envKey]
		if !ok {
			continue
		}

		if err := setFieldValue(field, rawValue, envKey); err != nil {
			return err
		}
	}

	return nil
}

func getEnvKey(field reflect.StructField) string {
	// Priority: env tag
	if tag := field.Tag.Get("env"); tag != "" {
		return tag
	}

	// Fallback: json tag
	if tag := field.Tag.Get("json"); tag != "" {
		name := strings.Split(tag, ",")[0]
		if name != "-" {
			return name
		}
	}

	// Fallback: field name
	return field.Name
}

func setFieldValue(field reflect.Value, raw string, key string) error {
	fieldType := field.Type()

	// Do not decode pointer-to-struct here.
	// Nested pointer structs are handled in fillStruct.
	if field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct {
		return nil
	}

	// Support pointer primitive fields:
	// *string, *int, *bool, etc.
	if field.Kind() == reflect.Ptr {
		elem := reflect.New(fieldType.Elem())

		if err := setFieldValue(elem.Elem(), raw, key); err != nil {
			return err
		}

		field.Set(elem)
		return nil
	}

	if fieldType == reflect.TypeOf(time.Duration(0)) {
		duration, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid duration for %s: %w", key, err)
		}

		field.SetInt(int64(duration))
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(raw)

	case reflect.Bool:
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid bool for %s: %w", key, err)
		}
		field.SetBool(value)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid int for %s: %w", key, err)
		}
		field.SetInt(value)

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid uint for %s: %w", key, err)
		}
		field.SetUint(value)

	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, field.Type().Bits())
		if err != nil {
			return fmt.Errorf("invalid float for %s: %w", key, err)
		}
		field.SetFloat(value)

	case reflect.Slice:
		return setSliceValue(field, raw, key)

	case reflect.Map:
		return setMapValue(field, raw, key)

	default:
		return fmt.Errorf("unsupported type for %s: %s", key, field.Kind())
	}

	return nil
}

func setMapValue(field reflect.Value, raw string, key string) error {
	if field.Type().Key().Kind() != reflect.String {
		return fmt.Errorf("unsupported map key type for %s: %s", key, field.Type().Key().Kind())
	}

	if field.Type().Elem().Kind() != reflect.String {
		return fmt.Errorf("unsupported map value type for %s: %s", key, field.Type().Elem().Kind())
	}

	mapValue := reflect.MakeMap(field.Type())

	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return fmt.Errorf("invalid map json for %s: %w", key, err)
	}

	for k, v := range parsed {
		mapValue.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
	}

	field.Set(mapValue)
	return nil
}

func setSliceValue(field reflect.Value, raw string, key string) error {
	elemKind := field.Type().Elem().Kind()

	parts := strings.Split(raw, ",")
	slice := reflect.MakeSlice(field.Type(), 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		elem := reflect.New(field.Type().Elem()).Elem()

		switch elemKind {
		case reflect.String:
			elem.SetString(part)

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			value, err := strconv.ParseInt(part, 10, elem.Type().Bits())
			if err != nil {
				return fmt.Errorf("invalid int slice value for %s: %w", key, err)
			}
			elem.SetInt(value)

		case reflect.Bool:
			value, err := strconv.ParseBool(part)
			if err != nil {
				return fmt.Errorf("invalid bool slice value for %s: %w", key, err)
			}
			elem.SetBool(value)

		default:
			return fmt.Errorf("unsupported slice type for %s: []%s", key, elemKind)
		}

		slice = reflect.Append(slice, elem)
	}

	field.Set(slice)
	return nil
}

func Load[T any](args *Args) (*T, error) {
	if args == nil {
		args = &Args{}
	}

	if args.Validate == nil {
		args.Validate = validator.New(
			validator.WithRequiredStructEnabled(),
		)
	}

	var err error

	if args.FileName != "" {
		log.Println("Loading env from file: ", args.FileName)
		if err := godotenv.Load(args.FileName); err != nil {
			return nil, err
		}
	}

	config, err := decodeEnvToStruct[T](loadFromEnv())
	if err != nil {
		return nil, err
	}

	if err := args.Validate.Struct(config); err != nil {
		return nil, err
	}

	return config, nil
}
