package preferences

import (
	"reflect"
	"testing"

	"github.com/longregen/alicia/api/domain"
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

func TestDefaultsMatchesDomain(t *testing.T) {
	// Fields that only exist in domain (not in defaults)
	domainOnly := map[string]bool{
		"UserID":    true,
		"CreatedAt": true,
		"UpdatedAt": true,
	}

	defaultsType := reflect.TypeOf(Defaults{})
	domainType := reflect.TypeOf(domain.UserPreferences{})

	// Build map of domain fields
	domainFields := make(map[string]reflect.Type)
	for i := 0; i < domainType.NumField(); i++ {
		f := domainType.Field(i)
		if !domainOnly[f.Name] {
			domainFields[f.Name] = f.Type
		}
	}

	// Check every Defaults field exists in domain
	for i := 0; i < defaultsType.NumField(); i++ {
		f := defaultsType.Field(i)
		domainFieldType, ok := domainFields[f.Name]
		if !ok {
			t.Errorf("field %s in Defaults missing from domain.UserPreferences - add to domain model and database", f.Name)
			continue
		}
		delete(domainFields, f.Name)

		// Check types are compatible (allow *int vs int for nullable fields)
		if !typesCompatible(f.Type, domainFieldType) {
			t.Errorf("field %s type mismatch: Defaults has %v, domain has %v", f.Name, f.Type, domainFieldType)
		}
	}

	// Warn about domain fields not in defaults (may be intentional)
	for name := range domainFields {
		t.Errorf("field %s in domain.UserPreferences missing from Defaults - add to defaults.json and Defaults struct", name)
	}
}

func typesCompatible(a, b reflect.Type) bool {
	if a == b {
		return true
	}
	// Allow int in Defaults to match *int in domain (nullable)
	if a.Kind() == reflect.Int && b.Kind() == reflect.Ptr && b.Elem().Kind() == reflect.Int {
		return true
	}
	return false
}
