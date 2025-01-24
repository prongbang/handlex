package main

import (
	"fmt"
	"github.com/dreamph/fibercore"
	"github.com/gofiber/fiber/v2"
	errs "github.com/pkg/errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
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
			return c.Status(http.StatusOK).JSON(data)
		},
		ResponseError: func(c *fiber.Ctx, requestOption *RequestOption, err error) error {
			res := &ErrorResponse{
				Status:     false,
				StatusCode: 500,
				Code:       "E00001",
				Message:    err.Error(),
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
	requestValidator := fibercore.NewRequestValidator()
	//requestValidator.RegisterValidation("my_validation", MyFunc)
	return fibercore.NewApiHandler[RequestInfo, RequestOption](NewApiResponseHandler(), &fibercore.ApiHandlerOptions[RequestInfo, RequestOption]{
		OnValidate: func(c *fiber.Ctx, requestOption *RequestOption, data any) error {
			if requestOption.EnableValidate {
				err := requestValidator.Validate(data)
				if err != nil {
					return &AppError{ErrCode: "V0001", ErrMessage: err.Error()}
				}
				return nil
			}
			return nil
		},
		OnBefore: func(c *fiber.Ctx, requestOption *RequestOption) error {
			log.Println("OnBefore")
			return nil
		},
		GetRequestInfo: func(c *fiber.Ctx, requestOption *RequestOption) (*RequestInfo, error) {
			log.Println("GetRequestInfo")
			return &RequestInfo{
				Token: "my-token",
			}, nil
		},
		OnAfter: func(c *fiber.Ctx, requestOption *RequestOption) error {
			log.Println("OnAfter")
			return nil
		},
	})
}

type UploadRequest struct {
	Name  string                `form:"name"`
	File  *multipart.FileHeader `form:"file" validate:"allow-file-extensions=.go,allow-file-mime-types=text/plain:text/plain2"`
	File2 *multipart.FileHeader `form:"file2"`
}

type SimpleRequest struct {
	Name string `json:"name"`
}

func main() {
	apiHandler := NewNewApiHandler()
	app := fiber.New()
	app.Get("/", func(c *fiber.Ctx) error {
		return apiHandler.Do(c, nil, nil, func(ctx fibercore.Context[RequestInfo]) (interface{}, error) {
			return "Hi.", nil
		})
	})

	app.Get("/custom-status", func(c *fiber.Ctx) error {
		requestOptions := fibercore.WithRequestOptions(
			SuccessStatus(201),
		)
		return apiHandler.Do(c, nil, requestOptions, func(ctx fibercore.Context[RequestInfo]) (interface{}, error) {
			return "Hi.", nil
		})
	})

	app.Get("/error", func(c *fiber.Ctx) error {
		return apiHandler.Do(c, nil, nil, func(ctx fibercore.Context[RequestInfo]) (interface{}, error) {
			return nil, &AppError{ErrCode: "0001", ErrMessage: "Error"}
		})
	})

	app.Post("/simple", func(c *fiber.Ctx) error {
		request := &SimpleRequest{}
		return apiHandler.Do(c, request, nil, func(ctx fibercore.Context[RequestInfo]) (interface{}, error) {
			fmt.Println(request.Name)
			return request.Name, nil
		})
	})

	app.Post("/upload", func(c *fiber.Ctx) error {
		request := &UploadRequest{}
		requestOptions := fibercore.WithRequestOptions(
			EnableValidate(true),
		)
		return apiHandler.Do(c, request, requestOptions, func(ctx fibercore.Context[RequestInfo]) (interface{}, error) {
			fmt.Println(request.Name)
			if request.File != nil {
				fmt.Println(request.File.Filename)
			}
			if request.File2 != nil {
				fmt.Println(request.File2.Filename)
			}
			return "Success", nil
		})
	})

	err := app.Listen(":3000")
	if err != nil {
		log.Fatal(err)
	}
}

//curl -v -F name=cenery -F file=@api.go -F file2=@utils.go http://localhost:3000/upload
