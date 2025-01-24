## Basic Usage
Full Example [example](example)

```go
package main

import (
	"context"
	"fmt"
	"github.com/dreamph/fibercore"
	"github.com/gofiber/fiber/v2"
	errs "github.com/pkg/errors"
	"io"
	"log"
	"mime/multipart"
	"time"
)

type RequestInfo struct {
	Token string `json:"token"`
}

type RequestOption struct {
	EnableValidate bool
	SuccessStatus  int
}

func EnableValidate(enable bool) fibercore.RequestOptions[RequestOption] {
	return func(opts *RequestOption) {
		opts.EnableValidate = enable
	}
}

func SuccessStatus(successStatus int) fibercore.RequestOptions[RequestOption] {
	return func(opts *RequestOption) {
		opts.SuccessStatus = successStatus
	}
}

type ErrorResponse struct {
	Status        bool            `json:"status"`
	StatusCode    int             `json:"statusCode"`
	StatusMessage string          `json:"statusMessage"`
	Type          string          `json:"type"`
	Code          string          `json:"code"`
	Message       string          `json:"message"`
	ErrorMessage  string          `json:"errorMessage"`
	Time          time.Time       `json:"time" swaggertype:"string" format:"date-time"`
	Detail        string          `json:"detail"`
	ErrorData     *[]AppErrorData `json:"errorData"`
	Cause         error           `json:"-"`
}

type AppErrorData struct {
	Reference    string           `json:"reference"`
	ErrorDetails []AppErrorDetail `json:"errorDetails"`
}

type AppErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type StreamData struct {
	Data io.Reader
	Size int `json:"status"`
}

type AppError struct {
	ErrCode    string `json:"errCode"`
	ErrMessage string `json:"errMessage"`
}

func (e *AppError) Error() string {
	return e.ErrCode + ":" + e.ErrMessage
}

func NewApiResponseHandler() fibercore.ApiResponseHandler[RequestOption] {
	apiResponseHandler := fibercore.NewApiResponseHandler[RequestOption](&fibercore.ApiResponseHandlerOptions[RequestOption]{
		ResponseSuccess: func(c *fiber.Ctx, requestOption *RequestOption, data any) error {
			if requestOption.SuccessStatus > 0 {
				c = c.Status(requestOption.SuccessStatus)
			}

			streamData, ok := data.(*StreamData)
			if ok {
				if streamData.Size > 0 {
					return c.SendStream(streamData.Data, streamData.Size)
				} else {
					return c.SendStream(streamData.Data)
				}
			}
			return c.JSON(data)
		},
		ResponseError: func(c *fiber.Ctx, requestOption *RequestOption, err error) error {
			res := &ErrorResponse{
				Status:     false,
				StatusCode: 500,
			}

			var appError *AppError
			ok := errs.As(err, &appError)
			if ok {
				res.Code = appError.ErrCode
				res.Message = appError.ErrMessage
				res.StatusCode = 400
			}

			return c.Status(res.StatusCode).JSON(res)
		},
	})
	return apiResponseHandler
}

func NewNewApiHandler() fibercore.ApiHandler[RequestInfo, RequestOption] {
	return fibercore.NewApiHandler[RequestInfo, RequestOption](NewApiResponseHandler(), &fibercore.ApiHandlerOptions[RequestInfo, RequestOption]{
		OnBefore: func(c *fiber.Ctx, requestOption *RequestOption) error {
			log.Println("OnBefore")
			if requestOption.EnableValidate {
				log.Println("EnableValidate")
			}
			return nil
		},
		GetRequestInfo: func(c *fiber.Ctx, requestOption *RequestOption) (*RequestInfo, error) {
			log.Println("GetRequestInfo")
			return &RequestInfo{}, nil
		},
		OnAfter: func(c *fiber.Ctx, requestOption *RequestOption) error {
			log.Println("OnAfter")
			return nil
		},
	})
}

type UploadRequest struct {
	File  *multipart.FileHeader `form:"file"`
	File2 *multipart.FileHeader `form:"file2"`
}

type SimpleRequest struct {
	Name string `json:"name"`
}

func main() {
	apiHandler := NewNewApiHandler()
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return apiHandler.Do(c, nil, nil, func(ctx context.Context, requestInfo *RequestInfo) (interface{}, error) {
			return "Hi.", nil
		})
	})

	app.Get("/custom-status", func(c *fiber.Ctx) error {
		requestOptions := fibercore.WithRequestOptions(
			SuccessStatus(201),
		)
		return apiHandler.Do(c, nil, requestOptions, func(ctx context.Context, requestInfo *RequestInfo) (interface{}, error) {
			return "Hi.", nil
		})
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		return apiHandler.Do(c, nil, nil, func(ctx context.Context, requestInfo *RequestInfo) (interface{}, error) {
			return nil, &AppError{ErrCode: "0001", ErrMessage: "Error"}
		})
	})

	app.Post("/simple", func(c *fiber.Ctx) error {
		request := &SimpleRequest{}
		return apiHandler.Do(c, request, nil, func(ctx context.Context, requestInfo *RequestInfo) (interface{}, error) {
			fmt.Println(request.Name)
			return request.Name, nil
		})
	})

	app.Post("/upload", func(c *fiber.Ctx) error {
		request := &UploadRequest{}
		return apiHandler.Do(c, request, nil, func(ctx context.Context, requestInfo *RequestInfo) (interface{}, error) {
			fmt.Println(request.File.Filename)
			fmt.Println(request.File2.Filename)
			return request.File.Filename + ", " + request.File2.Filename, nil
		})
	})

	err := app.Listen(":3000")
	if err != nil {
		log.Fatal(err)
	}
}

//curl -v -F name=cenery -F file=@api.go -F file2=@utils.go http://localhost:3000/upload

```


Buy Me a Coffee
=======
[!["Buy Me A Coffee"](https://www.buymeacoffee.com/assets/img/custom_images/orange_img.png)](https://www.buymeacoffee.com/dreamph)
