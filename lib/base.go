package lib

import (
	"time"
)

//状态标识
const (
	STATUS_ORIGINAL uint32 = 0 //原始
	STATUS_STARTED  uint32 = 1 //开始
	STATUS_STOPPED  uint32 = 2 //停止
)

// RetCode 表示结果代码的类型。
type RetCode int

// 保留 1 ~ 1000 给载荷承受方使用。
const (
	RET_CODE_SUCCESS              RetCode = 0    // 成功。
	RET_CODE_WARNING_CALL_TIMEOUT         = 1001 // 调用超时警告。
	RET_CODE_ERROR_CALL                   = 2001 // 调用错误。
	RET_CODE_ERROR_RESPONSE               = 2002 // 响应内容错误。
	RET_CODE_ERROR_CALEE                  = 2003 // 被调用方（被测软件）的内部错误。
	RET_CODE_FATAL_CALL                   = 3001 // 调用过程中发生了致命错误！
)

// GetRetCodePlain 会依据结果代码返回相应的文字解释。
func GetRetCodePlain(code RetCode) string {
	var codePlain string
	switch code {
	case RET_CODE_SUCCESS:
		codePlain = "Success"
	case RET_CODE_WARNING_CALL_TIMEOUT:
		codePlain = "Call Timeout Warning"
	case RET_CODE_ERROR_CALL:
		codePlain = "Call Error"
	case RET_CODE_ERROR_RESPONSE:
		codePlain = "Response Error"
	case RET_CODE_ERROR_CALEE:
		codePlain = "Callee Error"
	case RET_CODE_FATAL_CALL:
		codePlain = "Call Fatal Error"
	default:
		codePlain = "Unknown result code"
	}
	return codePlain
}

type Caller interface {
	BuildReq() RawReq                                         //创建请求
	Call(req []byte, timeoutNS time.Duration) ([]byte, error) //调用
	CheckResp(rawReq RawReq, rawResp RawResp) *CallRequst     //检查请求
}

type Sender interface {
	// 启动载荷发生器。
	Start() bool
	// 停止载荷发生器。
	// 第一个结果值代表已发载荷总数，且仅在第二个结果值为true时有效。
	// 第二个结果值代表是否成功将载荷发生器转变为已停止状态。
	Stop() bool
	// 获取状态。
	Status() uint32
}

//定义响应结果
type CallRequst struct {
	ID     int64         //ID
	Req    RawReq        //原生请求
	Resp   RawResp       //原生响应
	Code   RetCode       //响应代码
	Msg    string        //备注
	Timeup time.Duration //耗费时长
}

//原生请求
type RawReq struct {
	Id  int64
	Req []byte
}

//原生响应
type RawResp struct {
	ID     int64
	Resp   []byte
	Err    error
	Timeup time.Duration
}
