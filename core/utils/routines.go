package utils

import (
	"os"
	"path/filepath"
	"reflect"
)

func CleanTempRuntimes(pattern string) error {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if info.IsDir() {
			os.RemoveAll(path)
		}
	}
	return nil
}

func ExistsMatchingDirs(pattern, exclude string) (bool, error) {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return false, err
	}

	for _, path := range matches {
		if filepath.Clean(path) == filepath.Clean(exclude) {
			continue
		}
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return true, nil
		}
	}
	return false, nil
}

func IndexPaths(runDir string) (*map[string]string, error) {
	indexed := make(map[string]string)

	err := filepath.Walk(runDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(runDir, path)
		if err != nil {
			return err
		}

		indexed[relPath] = path
		return nil
	})

	if err != nil {
		return nil, err
	}

	return &indexed, nil
}

func IsFullyInitialized(i any) bool {
	v := reflect.ValueOf(i).Elem()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)

		switch field.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map, reflect.Chan, reflect.Func:
			if field.IsNil() {
				return false
			}
		case reflect.String:
			if field.String() == "" {
				return false
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if field.Int() == 0 {
				return false
			}
		case reflect.Bool:
			if !field.Bool() {
				return false
			}
		}
	}
	return true
}
