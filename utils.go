package fibercore

import (
	"github.com/gofiber/fiber/v2"
	"mime/multipart"
	"net/http"
	"reflect"
	"strings"
	"sync"
)

var (
	multipartFieldCache sync.Map
)

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

func isMultipartFileHeaderType(field reflect.StructField) bool {
	return field.Type == multipartFileHeaderType || field.Type == multipartFileHeaderPtrType
}

func cacheMultipartFieldCache(targetType reflect.Type) MultipartFieldCache {
	if cached, ok := multipartFieldCache.Load(targetType); ok {
		return cached.(MultipartFieldCache)
	}

	fields := make([]MultipartField, 0)
	for i := 0; i < targetType.NumField(); i++ {
		field := targetType.Field(i)
		if isMultipartFileHeaderType(field) {
			formTag := field.Tag.Get("form")
			if formTag != "" {
				fields = append(fields, MultipartField{
					Name:     formTag,
					FieldIdx: i,
				})
			}
		}
	}

	cache := MultipartFieldCache{
		RequiredMultipart: len(fields) > 0,
		Fields:            fields,
	}

	multipartFieldCache.Store(targetType, cache)
	return cache
}

func MultipartBodyParser(c *fiber.Ctx, targetPtr interface{}) error {
	v := reflect.ValueOf(targetPtr).Elem()
	t := v.Type()

	cache := cacheMultipartFieldCache(t)
	if !cache.RequiredMultipart {
		return nil
	}

	for _, field := range cache.Fields {
		file, err := c.FormFile(field.Name)
		if err != nil {
			return err
		}
		v.Field(field.FieldIdx).Set(reflect.ValueOf(file))
	}

	return nil
}

func IsMultipartForm(c *fiber.Ctx) bool {
	return strings.Contains(c.Get(fiber.HeaderContentType), fiber.MIMEMultipartForm)
}

func GetFileMimeType(fileHeader *multipart.FileHeader) (string, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer func(file multipart.File) {
		_ = file.Close()
	}(file)

	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}

	mimeType := http.DetectContentType(buffer)
	if strings.Contains(mimeType, ";") {
		return strings.Split(mimeType, ";")[0], nil
	}
	return mimeType, nil
}
