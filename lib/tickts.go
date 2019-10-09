//goroutine池的实现
package lib

import (
	"errors"
	"fmt"
)

//定义g池的类型
type myTickets struct {
	ticketTotol  uint32    //池总量
	ticketCh     chan byte //现有票数
	ticketStatus bool      //池状态
}

//定义g池的接口
type Gotickets interface {
	// 拿走一张票。
	Take()
	// 归还一张票。
	Return()
	// 票池是否已被激活。
	Active() bool
	// 票的总数。
	Total() uint32
	// 剩余的票数。
	Remainder() uint32
}

//初始化方法
func NewGoTickets(totol uint32) (Gotickets, error) {
	g := &myTickets{}
	if !g.init(totol) {
		return nil, errors.New(fmt.Sprintf("goutine池初始化错误,容量为%d", totol))
	} else {
		return g, nil
	}
}

//初始化池数据
func (g *myTickets) init(totol uint32) bool {

	if g.ticketStatus == false {
		return false
	}
	if totol == 0 {
		return false
	}

	ch := make(chan byte, totol)
	for i := 0; i < int(totol); i++ {
		ch <- 1
	}

	g.ticketTotol = totol
	g.ticketCh = ch
	g.ticketStatus = true

	return true
}
func (g *myTickets) Take() {
	<-g.ticketCh
}
func (g *myTickets) Return() {
	g.ticketCh <- 1
}
func (g *myTickets) Active() bool {
	return g.Active()
}
func (g *myTickets) Total() uint32 {
	return g.Total()
}
func (g *myTickets) Remainder() uint32 {
	return uint32(len(g.ticketCh))
}
