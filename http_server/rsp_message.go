package httpserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// 返回错误信息
func RspError(w http.ResponseWriter, err error) {
	var rspdata = make(map[string]interface{})
	w.Header().Set("Content-Type", "application/json")
	rspdata["code"] = 400
	rspdata["message"] = err.Error()
	data, err := json.Marshal(rspdata)
	if err != nil {
		log.Println(err.Error())
	}
	fmt.Fprint(w, string(data))
}

// 返回成功信息
func RspSuccess(w http.ResponseWriter, d interface{}) {
	var rspdata = make(map[string]interface{})
	w.Header().Set("Content-Type", "application/json")
	rspdata["code"] = 200
	rspdata["message"] = "success"
	rspdata["data"] = d
	data, err := json.Marshal(rspdata)
	if err != nil {
		log.Println(err.Error())
	}
	fmt.Fprint(w, string(data))
}
