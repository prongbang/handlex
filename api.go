package handlex

import (
	"context"
	"io"
	"mime/multipart"
	"net/http"
)

type Framework interface {
	Method() string
	UserContext() context.Context
	SendString(statusCode int, text string) error
	SendStream(stream io.Reader, size ...int) error
	JSON(data interface{}) error
	BodyParser(out interface{}) error
	FormFile(key string) (*multipart.FileHeader, error)
	Get(key string, defaultValue ...string) string
	Status(status int)
}

type RequestOptions[T any] func(opts *T)

func WithRequestOptions[T any](opts ...RequestOptions[T]) *T {
	var opt T
	if opts == nil {
		return &opt
	}
	for _, o := range opts {
		o(&opt)
	}
	return &opt
}

type DoFunc[RequestInfo any] func(ctx *Context[RequestInfo]) (interface{}, error)

type apiHandler[Fw Framework, RequestInfo any, RequestOption any] struct {
	apiResponseHandler ApiResponseHandler[Framework, RequestOption]
	options            *ApiHandlerOptions[Framework, RequestInfo, RequestOption]
}

type ApiHandlerOptions[Fw Framework, RequestInfo any, RequestOption any] struct {
	OnBefore       func(c Framework, requestOption *RequestOption) error
	GetRequestInfo func(c Framework, requestOption *RequestOption) (*RequestInfo, error)
	OnAfter        func(c Framework, requestOption *RequestOption) error
}

func NewApiHandler[Fw Framework, RequestInfo any, RequestOption any](apiResponseHandler ApiResponseHandler[Framework, RequestOption], options *ApiHandlerOptions[Framework, RequestInfo, RequestOption]) ApiHandler[Framework, RequestInfo, RequestOption] {
	return &apiHandler[Framework, RequestInfo, RequestOption]{
		apiResponseHandler: apiResponseHandler,
		options:            options,
	}
}

type ApiHandler[Fw Framework, RequestInfo any, RequestOption any] interface {
	Do(c Framework, requestPtr interface{}, requestOption *RequestOption, doFunc DoFunc[RequestInfo]) error
}

func (h *apiHandler[Framework, RequestInfo, RequestOption]) defaultRequestOptionIfNull(requestOption *RequestOption) *RequestOption {
	if requestOption != nil {
		return requestOption
	}

	var opt RequestOption
	return &opt
}

func (h *apiHandler[Framework, RequestInfo, RequestOption]) Do(c Framework, requestPtr any, requestOption *RequestOption, doFunc DoFunc[RequestInfo]) error {
	requestOption = h.defaultRequestOptionIfNull(requestOption)

	err := h.options.OnBefore(c, requestOption)
	if err != nil {
		return h.apiResponseHandler.ResponseError(c, requestOption, err)
	}

	_, err = h.bodyParserIfRequired(c, requestOption, requestPtr)
	if err != nil {
		return h.apiResponseHandler.ResponseError(c, requestOption, err)
	}

	requestInfo, err := h.options.GetRequestInfo(c, requestOption)
	if err != nil {
		return h.apiResponseHandler.ResponseError(c, requestOption, err)
	}

	data, err := doFunc(&Context[RequestInfo]{Context: c.UserContext(), RequestInfo: requestInfo})
	if err != nil {
		return h.apiResponseHandler.ResponseError(c, requestOption, err)
	}

	err = h.options.OnAfter(c, requestOption)
	if err != nil {
		return h.apiResponseHandler.ResponseError(c, requestOption, err)
	}

	return h.apiResponseHandler.ResponseSuccess(c, requestOption, data)
}

func (h *apiHandler[Framework, RequestInfo, RequestOption]) bodyParserIfRequired(c Framework, requestOption *RequestOption, requestPtr any) (bool, error) {
	if c.Method() == http.MethodGet {
		return false, nil
	}

	if requestPtr == nil {
		return false, nil
	}

	err := c.BodyParser(requestPtr)
	if err != nil {
		return false, h.apiResponseHandler.ResponseError(c, requestOption, err)
	}

	if IsMultipartForm(c) {
		err = MultipartBodyParser(c, requestPtr)
		if err != nil {
			return false, h.apiResponseHandler.ResponseError(c, requestOption, err)
		}
	}

	return true, nil
}

type ResponseData struct {
	Data interface{}
	Err  error
}

type Context[RequestInfo any] struct {
	Context     context.Context
	RequestInfo *RequestInfo
}

type ApiResponseHandlerOptions[Fw Framework, RequestOption any] struct {
	ResponseSuccess func(c Framework, requestOption *RequestOption, data any) error
	ResponseError   func(c Framework, requestOption *RequestOption, err error) error
}

type ApiResponseHandler[Fw Framework, RequestOption any] interface {
	ResponseSuccess(c Framework, requestOption *RequestOption, data any) error
	ResponseError(c Framework, requestOption *RequestOption, err error) error
}

type responseHandler[Fw Framework, RequestOption any] struct {
	options *ApiResponseHandlerOptions[Framework, RequestOption]
}

func (r responseHandler[Framework, RequestOption]) ResponseError(c Framework, requestOption *RequestOption, err error) error {
	if r.options.ResponseError != nil {
		return r.options.ResponseError(c, requestOption, err)
	}
	return c.SendString(500, err.Error())
}

func (r responseHandler[Framework, RequestOption]) ResponseSuccess(c Framework, requestOption *RequestOption, data any) error {
	if r.options.ResponseSuccess != nil {
		return r.options.ResponseSuccess(c, requestOption, data)
	}
	return c.JSON(data)
}

func NewApiResponseHandler[Fw Framework, RequestOption any](options *ApiResponseHandlerOptions[Framework, RequestOption]) ApiResponseHandler[Framework, RequestOption] {
	return &responseHandler[Framework, RequestOption]{
		options: options,
	}
}
