package sender

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"sender/lib"
	"strings"
	"sync/atomic"
	"time"
)

//定义载荷发生器结构
type mySender struct {
	caller      lib.Caller           // 调用器。
	timeoutNS   time.Duration        // 处理超时时间，单位：纳秒。
	lps         uint32               // 每秒载荷量。
	durationNS  time.Duration        // 负载持续时间，单位：纳秒。
	concurrency uint32               // 载荷并发量。
	tickets     lib.Gotickets        // Goroutine票池。
	ctx         context.Context      // 上下文。
	cancelFunc  context.CancelFunc   // 取消函数。
	callCount   int64                // 调用计数。
	status      uint32               // 状态。
	resultCh    chan *lib.CallRequst // 调用结果通道。
}

//输入的结构体
type ParamSet struct {
	Caller     lib.Caller           // 调用器。
	TimeoutNS  time.Duration        // 响应超时时间，单位：纳秒。
	LPS        uint32               // 每秒载荷量。
	DurationNS time.Duration        // 负载持续时间，单位：纳秒。
	ResultCh   chan *lib.CallRequst // 调用结果通道。
}

func (pset *ParamSet) Check() bool {
	var errMsgs []string

	if pset.Caller == nil {
		errMsgs = append(errMsgs, "Invalid caller!")
	}
	if pset.TimeoutNS == 0 {
		errMsgs = append(errMsgs, "Invalid timeoutNS!")
	}
	if pset.LPS == 0 {
		errMsgs = append(errMsgs, "Invalid lps(load per second)!")
	}
	if pset.DurationNS == 0 {
		errMsgs = append(errMsgs, "Invalid durationNS!")
	}
	if pset.ResultCh == nil {
		errMsgs = append(errMsgs, "Invalid result channel!")
	}
	if errMsgs != nil {
		errMsg := strings.Join(errMsgs, " ")
		fmt.Printf("%s", errMsg)
		return false
	}
	fmt.Printf("Passed.\n (timeoutNS=%s, lps=%d, durationNS=%s)", pset.TimeoutNS, pset.LPS, pset.DurationNS)
	return true
}

func NewSender(pset ParamSet) (lib.Sender, error) {
	fmt.Printf("New a sender")
	if !pset.Check() {
		return nil, errors.New("参数出错.")
	}
	gen := &mySender{
		caller:     pset.Caller,
		timeoutNS:  pset.TimeoutNS,
		lps:        pset.LPS,
		durationNS: pset.DurationNS,
		status:     lib.STATUS_ORIGINAL,
		resultCh:   pset.ResultCh,
	}
	if err := gen.init(); err != nil {
		return nil, err
	}
	return gen, nil
}

func (ms *mySender) init() error {
	var buf bytes.Buffer
	buf.WriteString("Initializing the load generator...")
	// 载荷的并发量 ≈ 载荷的响应超时时间 / 载荷的发送间隔时间
	var total64 = int64(ms.timeoutNS)/int64(1e9/ms.lps) + 1
	if total64 > math.MaxInt32 {
		total64 = math.MaxInt32
	}
	ms.concurrency = uint32(total64)
	tickets, err := lib.NewGoTickets(ms.concurrency)
	if err != nil {
		return err
	}

	ms.tickets = tickets

	buf.WriteString(fmt.Sprintf("Done. (concurrency=%d)", ms.concurrency))
	fmt.Printf("%s", buf.String())
	return nil
}

//发起一次调用
func (ms *mySender) callOne(req *lib.RawReq) *lib.RawResp {
	atomic.AddInt64(&ms.callCount, 1)
	if req == nil {
		return &lib.RawResp{ID: -1, Err: errors.New("请求为空，无法开启调用")}
	}

	strtTime := time.Now().UnixNano()
	result, err := ms.caller.Call(req.Req, ms.timeoutNS)
	endTime := time.Now().UnixNano()

	duration := time.Duration(strtTime - endTime)

	if err != nil {
		return &lib.RawResp{
			ID:     req.Id,
			Err:    errors.New(fmt.Sprintf("Call one error :%s", err)),
			Timeup: duration,
		}
	} else {
		return &lib.RawResp{
			ID:     req.Id,
			Timeup: duration,
			Resp:   result,
		}
	}

}
func (ms *mySender) sendResult(result *lib.CallRequst) bool {
	if atomic.LoadUint32(&ms.status) != lib.STATUS_STARTED {
		fmt.Printf("接受结果失败。sender已经关闭了\n")
		return false
	}

	select {
	case ms.resultCh <- result:
		return true
	default:
		fmt.Printf("接受结果失败。通道已经满了\n")
		return false
	}

}

//不断的发送请求给服务器
func (ms *mySender) sendload(rate <-chan time.Time) {

	for {

		select {
		case <-ms.ctx.Done():
			close(ms.resultCh)
			fmt.Printf("系统正在关闭")
			return
		default:
		}
		if ms.lps > 0 {
			select {
			case <-rate:
				ms.asyncCall()
			case <-ms.ctx.Done():
				close(ms.resultCh)
				fmt.Printf("系统正在关闭")
				return
			}
		}

	}

}

//异步的发起多次调用
func (ms *mySender) asyncCall() {
	ms.tickets.Take()
	go func() {
		defer func() {
			if p := recover(); p != nil {
				err, ok := interface{}(p).(error)
				var errMsg string
				if ok {
					errMsg = fmt.Sprintf("Async Call Panic! (error: %s)", err)
				} else {
					errMsg = fmt.Sprintf("Async Call Panic! (clue: %#v)", p)
				}
				fmt.Printf("%s", errMsg)
				result := &lib.CallRequst{
					ID:   -1,
					Code: lib.RET_CODE_FATAL_CALL,
					Msg:  errMsg}
				ms.sendResult(result)
			}
			ms.tickets.Return()
		}()
		//初始化调用信息
		rawreq := ms.caller.BuildReq()
		//调用状态  0 初始  1超时  2调用完成
		var callStatus uint32
		//定时器检测是否超时。
		timer := time.AfterFunc(ms.timeoutNS, func() {
			if atomic.CompareAndSwapUint32(&callStatus, 0, 1) {
				result := &lib.CallRequst{
					ID:     rawreq.Id,
					Req:    rawreq,
					Code:   lib.RET_CODE_WARNING_CALL_TIMEOUT,
					Msg:    fmt.Sprintf("Timeout! (expected: < %v)", ms.timeoutNS),
					Timeup: ms.timeoutNS,
				}
				ms.sendResult(result)
			} else {
				return
			}
		})
		if !atomic.CompareAndSwapUint32(&callStatus, 0, 2) {
			return
		}
		timer.Stop()
		rawresq := ms.callOne(&rawreq)
		var result *lib.CallRequst

		if rawresq.Err != nil {
			result = &lib.CallRequst{
				ID:     rawresq.ID,
				Req:    rawreq,
				Code:   lib.RET_CODE_ERROR_CALL,
				Msg:    rawresq.Err.Error(),
				Timeup: rawresq.Timeup,
			}
		} else {
			result = ms.caller.CheckResp(rawreq, *rawresq)
			result.Timeup = rawresq.Timeup
		}

		ms.sendResult(result)

	}()

}

func (ms *mySender) Start() bool {
	fmt.Printf("正在启动载荷发射器...")
	if !atomic.CompareAndSwapUint32(&ms.status, lib.STATUS_ORIGINAL, lib.STATUS_STARTED) {
		fmt.Printf("启动失败..")
		return false
	}

	var rate <-chan time.Time

	if ms.lps > 0 {

		rate = time.Tick(time.Duration(1e9 / ms.lps))
	}

	ms.ctx, ms.cancelFunc = context.WithTimeout(context.Background(), ms.durationNS)
	ms.callCount = 0

	go func() {
		fmt.Printf("我要开始发送咯！..\n")
		ms.sendload(rate)
		fmt.Printf("一共调用%d次\n", ms.callCount)

	}()
	return true
}

func (ms *mySender) Stop() bool {
	if !atomic.CompareAndSwapUint32(&ms.status, lib.STATUS_STARTED, lib.STATUS_STOPPED) {
		fmt.Printf("已经停止！")
		return false
	}

	ms.cancelFunc()
	close(ms.resultCh)
	return true

}
func (ms *mySender) Status() uint32 {
	return atomic.LoadUint32(&ms.status)
}
