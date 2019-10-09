package testhelper

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type loginreq struct {
	OpName string
	OpNum  []int
}

type server struct {
}

func NewServer() *server {
	s := server{}
	return &s
}

func (s *server) Init() {
	http.HandleFunc("/op0", operation)
	http.ListenAndServe("0.0.0.0:8090", nil)
}

func operation(w http.ResponseWriter, r *http.Request) {
	req := &loginreq{}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("读取错误:%v", err)
		return
	}
	err1 := json.Unmarshal(data, req)
	w.Write(data)

	if err1 != nil {
		fmt.Println("error:", err)
	}
	fmt.Printf("%s\n", req.OpName)
	for _, v := range req.OpNum {
		fmt.Printf("%d\n", v)
	}

}
