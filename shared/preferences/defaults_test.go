package preferences

import (
	"reflect"
	"testing"
)

func TestDefaultsAllFieldsSet(t *testing.T) {
	d := Get()
	v := reflect.ValueOf(d)
	typ := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := typ.Field(i)

		// Skip bools - false is a valid default
		if field.Kind() == reflect.Bool {
			continue
		}

		if field.IsZero() {
			t.Errorf("field %s has zero value - missing from defaults.json?", fieldType.Name)
		}
	}
}
