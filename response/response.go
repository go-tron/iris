package response

import (
	"github.com/go-tron/iris/baseContext"
)

func New() *Response {
	return &Response{}
}

type Response struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	System  bool        `json:"system,omitempty"`
	Chain   string      `json:"chain,omitempty"`
	Rid     interface{} `json:"rid,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func (r *Response) New(code string, msg string, data ...interface{}) baseContext.Response {
	return (&Response{Code: code, Message: msg}).WithData(data...)
}

func (r *Response) Success(data ...interface{}) baseContext.Response {
	if len(data) == 0 {
		return r.New("00", "")
	} else if len(data) == 1 {
		switch v := data[0].(type) {
		case baseContext.Response:
			return v
		default:
			return r.New("00", "", data[0])
		}
	} else {
		return r.New("00", "", data...)
	}
}

func (r *Response) Error(code string, msg string, data ...interface{}) baseContext.Response {
	return r.New(code, msg, data...)
}

func (r *Response) WithCode(code string) baseContext.Response {
	r.Code = code
	return r
}

func (r *Response) WithMessage(message string) baseContext.Response {
	r.Message = message
	return r
}

func (r *Response) WithData(data ...interface{}) baseContext.Response {
	if len(data) == 1 {
		r.Data = data[0]
	} else if len(data) > 1 {
		r.Data = data
	}
	return r
}

func (r *Response) WithSystem() baseContext.Response {
	r.System = true
	return r
}

func (r *Response) WithChain(chain string) baseContext.Response {
	r.Chain = chain
	return r
}

func (r *Response) WithRid(rid string) baseContext.Response {
	r.Rid = rid
	return r
}

func (r *Response) ContentType() string {
	return "json"
}

func (r *Response) Content() interface{} {
	return r
}
