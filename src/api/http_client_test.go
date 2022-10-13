package api

import (
	"log"
	"testing"
)

func TestTphttp_Post(t *testing.T) {
	var req = make(map[string]string)
	req["AccessToken"] = "123456"
	rsp, _ := PostJson("http://127.0.0.1:9999/api/gateway/config", req)
	log.Println(rsp)
}
