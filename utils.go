package fibercore

import (
	"github.com/gofiber/fiber/v2"
	"mime/multipart"
	"reflect"
	"strings"
	"sync"
)

var multipartFieldCache sync.Map

func isMultipartFileHeader(field reflect.StructField) bool {
	if field.Type == reflect.TypeOf((*multipart.FileHeader)(nil)) {
		return true
	}
	if field.Type == reflect.TypeOf(multipart.FileHeader{}) {
		return true
	}
	return false
}

func cacheStructFields(targetType reflect.Type) MultipartCache {
	if cached, ok := multipartFieldCache.Load(targetType); ok {
		return cached.(MultipartCache)
	}

	var fields []structField
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)

		//fmt.Println(reflect.TypeOf(multipart.FileHeader{}))
		//fmt.Println(field.Type)
		if !isMultipartFileHeader(field) {
			continue
		}

		formTag := field.Tag.Get("form")
		if formTag != "" {
			fields = append(fields, structField{
				Name:     formTag,
				FieldIdx: i,
			})
		}
	}

	multipartCache := MultipartCache{
		RequiredMultipart: len(fields) > 0,
		Fields:            fields,
	}

	multipartFieldCache.Store(targetType, multipartCache)

	return multipartCache
}

type MultipartCache struct {
	RequiredMultipart bool
	Fields            []structField
}

type structField struct {
	Name     string // Form field name
	FieldIdx int    // Index of the field in the struct
}

func MultipartBodyParser(c *fiber.Ctx, targetPtr interface{}) error {
	v := reflect.ValueOf(targetPtr).Elem()
	t := v.Type()

	multipartCache := cacheStructFields(t)
	if !multipartCache.RequiredMultipart {
		return nil
	}

	for _, field := range multipartCache.Fields {
		fieldValue := v.Field(field.FieldIdx)
		file, err := c.FormFile(field.Name)
		if err != nil {
			return err
		}
		fieldValue.Set(reflect.ValueOf(file))
	}

	return nil
}

func IsMultipartForm(c *fiber.Ctx) bool {
	contentType := c.Get("Content-Type")
	return strings.Contains(contentType, "multipart/form-data")
}
