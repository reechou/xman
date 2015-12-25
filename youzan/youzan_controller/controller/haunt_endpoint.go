
package controller

import (
	"net/http"
	"encoding/json"
	"time"
	"fmt"
)

type HauntHttpLogic struct {
}

type HttpHandler func(rsp http.ResponseWriter, req *http.Request) (interface{}, error)

func (self *HauntHttpLogic) httpWrap(handler HttpHandler) func(rsp http.ResponseWriter, req *http.Request) {
	f := func(rsp http.ResponseWriter, req *http.Request) {
		url := req.URL.String()
		start := time.Now()
		defer func() {
			fmt.Printf("[httpWrap] http: request url[%s] use_time[%v]\n", url, time.Now().Sub(start))
		}()
		obj, err := handler(rsp, req)
	HAS_ERR:
		if err != nil {
			code := 500
			rsp.WriteHeader(code)
			rsp.Write([]byte("HTTP ERROR."))
			return
		}

		if obj != nil {
			var buf []byte
			buf, err = json.Marshal(obj)
			if err != nil {
				goto HAS_ERR
			}
			rsp.Header().Set("Content-Type", "application/json")
			rsp.Write(buf)
		}
	}
	return f
}

func (self *HauntHttpLogic) GetRoot(rsp http.ResponseWriter, req *http.Request) (interface{}, error) {
	req.ParseForm()
	list, err := Logic.GetRoot()
	if err != nil {
		rsp.WriteHeader(400)
		rsp.Write([]byte(err.Error()))
		return nil, nil
	}
	return list, nil
}

func (self *HauntHttpLogic) GetModules(rsp http.ResponseWriter, req *http.Request) (interface{}, error) {
	req.ParseForm()
	list, err := Logic.GetModules(req.Form["ns"][0])
	if err != nil {
		rsp.WriteHeader(400)
		rsp.Write([]byte(err.Error()))
		return nil, nil
	}
	return list, nil
}
