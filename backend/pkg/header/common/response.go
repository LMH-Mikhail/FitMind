package common

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BaseResponse是后端所有HTTP接口的统一JSON响应结构
// Data：业务数据，成功时通常有值，失败时通常为空
// Code：响应码，默认与HTTP状态码一致；必要时可以作为业务码使用
// Message：响应信息，成功默认为“success”，失败为错误原因
// Success：是否成功
// httpStatus：实际HTTP状态码，不输出到JSON
type BaseResponse struct {
	Data       any    `json:"data"`
	Code       int    `json:"code"`
	Message    string `json:"message"`
	Success    bool   `json:"success"`
	httpStatus int    `json:"-"`
}

// NewResponse创建默认成功响应
// 输入：无
// 输出：默认成功的BaseResponse指针
// 默认值：HTTP 200、code 200、message "success"、success true
func NewResponse() *BaseResponse {
	return &BaseResponse{
		Data:       nil,
		Code:       http.StatusOK,
		Message:    "success",
		Success:    true,
		httpStatus: http.StatusOK,
	}
}

// SetSuccess将响应重置为成功状态
// 输入：无
// 输出：修改当前BaseResponse
// 使用场景：controller需要显式覆盖之前的错误状态时调用
func (response *BaseResponse) SetSuccess() {
	response.Code = http.StatusOK
	response.Success = true
	response.Message = "success"
	response.httpStatus = http.StatusOK
}

// SetError设置错误响应
// 输入：code通常为HTTP状态码；message为错误原因
// 输出：修改当前BaseResponse
// 使用场景：HTTP状态码与响应体code一致时使用
func (response *BaseResponse) SetError(code int, message string) {
	response.Code = code
	response.Success = false
	response.Message = message
	response.httpStatus = code
}

// SetErrorWithHTTPStatus设置错误响应，并允许HTTP状态码与响应体code分离
// 输入：
// - httpStatus：实际HTTP状态码
// - code：JSON响应体中的业务码
// - message：错误原因
// 输出：修改当前BaseResponse
// 使用场景：后续如果引入业务错误码，但HTTP仍返回标准状态码，可以使用此方法
func (response *BaseResponse) SetErrorWithHTTPStatus(httpStatus int, code int, message string) {
	response.Code = code
	response.Success = false
	response.Message = message
	response.httpStatus = httpStatus
}

// Response将统一响应写入Gin HTTP上下文
// 输入：ctx为当前请求上下文
// 输出：向客户端写出JSON响应
// 规则：如果httpStatus未设置，则兜底使用Code；如果Code也未设置，则使用HTTP 200
func (response *BaseResponse) Response(ctx *gin.Context) {
	httpStatus := response.httpStatus
	if httpStatus == 0 {
		httpStatus = response.Code
	}
	if httpStatus == 0 {
		httpStatus = http.StatusOK
	}
	ctx.JSON(httpStatus, response)
}

// Page是分页列表接口的统一响应结构
// PageNum：当前页码，从1开始
// PageSize：每页数量
// TotalPage：总页数
// Total：总记录数
// List：当前页列表数据
type Page struct {
	PageNum   int `json:"pageNum"`
	PageSize  int `json:"pageSize"`
	TotalPage int `json:"totalPage"`
	Total     int `json:"total"`
	List      any `json:"list"`
}
