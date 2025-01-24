package fibercore

import (
	"github.com/gofiber/fiber/v2"
	"mime/multipart"
	"reflect"
	"strings"
	"sync"
)

var multipartFieldCache sync.Map

var (
	multipartFileHeaderType    = reflect.TypeFor[multipart.FileHeader]()
	multipartFileHeaderPtrType = reflect.TypeFor[*multipart.FileHeader]()
)

type MultipartFieldCache struct {
	RequiredMultipart bool
	Fields            []MultipartField
}

type MultipartField struct {
	Name     string // Form field name
	FieldIdx int    // Index of the field in the struct
}

func isMultipartFileHeader(field reflect.StructField) bool {
	if field.Type == multipartFileHeaderType {
		return true
	}
	if field.Type == multipartFileHeaderPtrType {
		return true
	}
	return false
}

func cacheMultipartFieldCache(targetType reflect.Type) MultipartFieldCache {
	if cached, ok := multipartFieldCache.Load(targetType); ok {
		return cached.(MultipartFieldCache)
	}

	var fields []MultipartField
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if !isMultipartFileHeader(field) {
			continue
		}

		formTag := field.Tag.Get("form")
		if formTag != "" {
			fields = append(fields, MultipartField{
				Name:     formTag,
				FieldIdx: i,
			})
		}
	}

	multipartCache := MultipartFieldCache{
		RequiredMultipart: len(fields) > 0,
		Fields:            fields,
	}

	multipartFieldCache.Store(targetType, multipartCache)

	return multipartCache
}

func MultipartBodyParser(c *fiber.Ctx, targetPtr interface{}) error {
	v := reflect.ValueOf(targetPtr).Elem()
	t := v.Type()

	cache := cacheMultipartFieldCache(t)
	if !cache.RequiredMultipart {
		return nil
	}

	for _, field := range cache.Fields {
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
	contentType := c.Get(fiber.HeaderContentType)
	return strings.Contains(contentType, fiber.MIMEMultipartForm)
}
