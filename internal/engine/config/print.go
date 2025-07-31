package config

import (
	"fmt"
	"reflect"
	"time"
)

func (c *Compositor) Print(v any) {
	c.printConfig(v, "  ")
}

func (c *Compositor) printConfig(v any, prefix string) {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		fieldName := fieldType.Name
		if tag, ok := fieldType.Tag.Lookup("mapstructure"); ok {
			if tag != "" {
				fieldName = tag
			}
		}

		if field.Kind() == reflect.Ptr {
			if field.IsNil() {
				fmt.Printf("%s%s: <nil>\n", prefix, fieldName)
				continue
			}
			field = field.Elem()
		}

		if field.Kind() == reflect.Struct {
			if field.Type() == reflect.TypeOf(time.Duration(0)) {
				duration := field.Interface().(time.Duration)
				fmt.Printf("%s%s: %s\n", prefix, fieldName, duration.String())
			} else {
				fmt.Printf("%s%s:\n", prefix, fieldName)
				c.printConfig(field.Addr().Interface(), prefix+"  ")
			}
		} else if field.Kind() == reflect.Slice {
			fmt.Printf("%s%s: %v\n", prefix, fieldName, field.Interface())
		} else {
			value := field.Interface()
			if field.Kind() == reflect.String {
				value = fmt.Sprintf("\"%s\"", value)
			}
			fmt.Printf("%s%s: %v\n", prefix, fieldName, value)
		}
	}
}
