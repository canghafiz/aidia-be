package helpers

import (
	"fmt"
	"log"
	"reflect"

	"github.com/go-playground/validator/v10"
)

func FatalError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func ErrValidator(request interface{}, validator *validator.Validate) error {
	val := reflect.ValueOf(request)

	// if slice
	if val.Kind() == reflect.Slice {
		for i := 0; i < val.Len(); i++ {
			item := val.Index(i).Interface()
			if err := validator.Struct(item); err != nil {
				return fmt.Errorf("validation failed at index %d: %w", i, err)
			}
		}
		return nil
	}

	// if single
	if err := validator.Struct(request); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	return nil
}
