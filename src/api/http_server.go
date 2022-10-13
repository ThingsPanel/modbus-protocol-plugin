package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/viper"
)

func HttpServer() {
	http.HandleFunc("/api/form/config", GetFormConfig)                      //设置访问的路由
	err := http.ListenAndServe(viper.GetString("http_server.address"), nil) //设置监听的端口
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}
func GetFormConfig(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	log.Println("【收到api请求】path", r.URL.Path)
	log.Println("scheme", r.URL.Scheme)
	var rsp = make(map[string]interface{})
	rsp["data"] = ReadFormConfig()
	data, err := json.Marshal(rsp)
	if err != nil {
		log.Println(err.Error())
	}
	fmt.Fprint(w, string(data)) //这个写入到w的是输出到客户端的
}

func ReadFormConfig() interface{} {
	filePtr, err := os.Open("./form_config.json")
	if err != nil {
		log.Println("文件打开失败...", err.Error())
		return nil
	}
	defer filePtr.Close()
	var info interface{}
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&info)
	if err != nil {
		log.Println("解码失败", err.Error())
		return info
	} else {
		log.Println("读取文件[form_config.json]成功...")
		return info
	}
}
