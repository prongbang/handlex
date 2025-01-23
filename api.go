package fibercore

import (
	"context"
	"github.com/gofiber/fiber/v2"
	"net/http"
)

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

type DoFunc[RequestInfo any] func(ctx context.Context, requestInfo *RequestInfo) (interface{}, error)

type apiHandler[RequestInfo any, RequestOption any] struct {
	apiResponseHandler ApiResponseHandler[RequestOption]
	options            *ApiHandlerOptions[RequestInfo, RequestOption]
}

type ApiHandlerOptions[RequestInfo any, RequestOption any] struct {
	OnBefore       func(c *fiber.Ctx, requestOption *RequestOption) error
	GetRequestInfo func(c *fiber.Ctx, requestOption *RequestOption) (*RequestInfo, error)
	OnAfter        func(c *fiber.Ctx, requestOption *RequestOption) error
}

func NewApiHandler[RequestInfo any, RequestOption any](apiResponseHandler ApiResponseHandler[RequestOption], options *ApiHandlerOptions[RequestInfo, RequestOption]) ApiHandler[RequestInfo, RequestOption] {
	return &apiHandler[RequestInfo, RequestOption]{
		apiResponseHandler: apiResponseHandler,
		options:            options,
	}
}

type ApiHandler[RequestInfo any, RequestOption any] interface {
	Do(c *fiber.Ctx, requestPtr interface{}, requestOption *RequestOption, doFunc DoFunc[RequestInfo]) error
}

func (h *apiHandler[RequestInfo, RequestOption]) defaultRequestOptionIfNull(requestOption *RequestOption) *RequestOption {
	if requestOption != nil {
		return requestOption
	}

	var opt RequestOption
	return &opt
}
func (h *apiHandler[RequestInfo, RequestOption]) Do(c *fiber.Ctx, requestPtr any, requestOption *RequestOption, doFunc DoFunc[RequestInfo]) error {
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

	data, err := doFunc(c.UserContext(), requestInfo)
	if err != nil {
		return h.apiResponseHandler.ResponseError(c, requestOption, err)
	}

	err = h.options.OnAfter(c, requestOption)
	if err != nil {
		return h.apiResponseHandler.ResponseError(c, requestOption, err)
	}

	return h.apiResponseHandler.ResponseSuccess(c, requestOption, data)
}

func (h *apiHandler[RequestInfo, RequestOption]) bodyParserIfRequired(c *fiber.Ctx, requestOption *RequestOption, requestPtr any) (bool, error) {
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

	return true, nil
}

type ResponseData struct {
	Data interface{}
	Err  error
}

type ApiResponseHandlerOptions[RequestOption any] struct {
	ResponseSuccess func(c *fiber.Ctx, requestOption *RequestOption, data any) error
	ResponseError   func(c *fiber.Ctx, requestOption *RequestOption, err error) error
}

type ApiResponseHandler[RequestOption any] interface {
	ResponseSuccess(c *fiber.Ctx, requestOption *RequestOption, data any) error
	ResponseError(c *fiber.Ctx, requestOption *RequestOption, err error) error
}

type responseHandler[RequestOption any] struct {
	options *ApiResponseHandlerOptions[RequestOption]
}

func (r responseHandler[RequestOption]) ResponseError(c *fiber.Ctx, requestOption *RequestOption, err error) error {
	if r.options.ResponseError != nil {
		return r.options.ResponseError(c, requestOption, err)
	}
	return c.Status(500).SendString(err.Error())
}

func (r responseHandler[RequestOption]) ResponseSuccess(c *fiber.Ctx, requestOption *RequestOption, data any) error {
	if r.options.ResponseSuccess != nil {
		return r.options.ResponseSuccess(c, requestOption, data)
	}
	return c.JSON(data)
}

func NewApiResponseHandler[RequestOption any](options *ApiResponseHandlerOptions[RequestOption]) ApiResponseHandler[RequestOption] {
	return &responseHandler[RequestOption]{
		options: options,
	}
}
