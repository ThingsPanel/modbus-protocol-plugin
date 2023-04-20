package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	server_map "tp-modbus/map"
	"tp-modbus/src/modbus_rtu"
	"tp-modbus/src/util"

	"github.com/spf13/viper"
)

func HttpServer() {
	http.HandleFunc("/api/form/config", GetFormConfig)                      //获取插件表单配置
	http.HandleFunc("/api/device/config/update", UpdateSubDeviceConfig)     //修改子设备配置
	http.HandleFunc("/api/device/config/add", AddSubDeviceConfig)           //新增子设备配置
	http.HandleFunc("/api/device/config/delete", DeleteSubDeviceConfig)     //删除子设备配置
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
	rsp["data"] = util.ReadFormConfig()
	data, err := json.Marshal(rsp)
	if err != nil {
		log.Println(err.Error())
	}
	fmt.Fprint(w, string(data)) //这个写入到w的是输出到客户端的
}

type UpdateSubDeviceConfigStruct struct {
	GateWayId    string
	DeviceId     string
	DeviceConfig server_map.Device
}

// 添加子设备配置
func AddSubDeviceConfig(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	log.Println("【收到api请求】path", r.URL.Path)
	log.Println("scheme", r.URL.Scheme)
	log.Println(r.Method)
	if r.Method != "POST" {
		//w.WriteHeader(201)
		return
	}
	var reqdata UpdateSubDeviceConfigStruct
	if err := json.NewDecoder(r.Body).Decode(&reqdata); err != nil {
		r.Body.Close()
		w.WriteHeader(400)
		return
	}
	log.Println("req json: ", reqdata)
	// 添加子设备需要将子设备配置添加到map中
	server_map.SubDeviceConfigMap[reqdata.DeviceId] = reqdata.DeviceConfig
	var rspdata = make(map[string]interface{})
	w.Header().Set("Content-Type", "application/json")
	rspdata["code"] = 200
	rspdata["message"] = "success"
	//校验表单
	if reqdata.DeviceConfig.FunctionCode == uint8(1) {
		if len(reqdata.DeviceConfig.Key) > int(reqdata.DeviceConfig.AddressNum) {
			rspdata["code"] = 400
			rspdata["message"] = "属性别名数量与线圈数量不匹配！"
		}
	}
	// 如果设备在线，启动子设备采集
	if gc, ok := server_map.GatewayChannelMap[reqdata.GateWayId]; ok {
		if len(gc) != 1 { //判断设备是否在线
			if server_map.GatewayConfigMap[reqdata.GateWayId].ProtocolType == "MODBUS_RTU" {
				go modbus_rtu.InitRTUGo(reqdata.GateWayId, reqdata.DeviceId)
			} else if server_map.GatewayConfigMap[reqdata.GateWayId].ProtocolType == "MODBUS_TCP" {
				go modbus_rtu.InitTCPGo(reqdata.GateWayId, reqdata.DeviceId)
			}
		}
	}
	data, err := json.Marshal(rspdata)
	if err != nil {
		log.Println(err.Error())
	}
	fmt.Fprint(w, string(data))
}

// 修改子设备配置
func UpdateSubDeviceConfig(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	log.Println("【收到api请求】path", r.URL.Path)
	log.Println("scheme", r.URL.Scheme)
	log.Println(r.Method)
	if r.Method != "POST" {
		//w.WriteHeader(201)
		return
	}
	var reqdata UpdateSubDeviceConfigStruct
	if err := json.NewDecoder(r.Body).Decode(&reqdata); err != nil {
		r.Body.Close()
		w.WriteHeader(400)
		return
	}
	log.Println("req json: ", reqdata)

	server_map.SubDeviceConfigMap[reqdata.DeviceId] = reqdata.DeviceConfig //修改子设备配置只需要修改map中的配置
	var rspdata = make(map[string]interface{})
	w.Header().Set("Content-Type", "application/json")
	rspdata["code"] = 200
	rspdata["message"] = "success"
	//校验表单
	if reqdata.DeviceConfig.FunctionCode == uint8(1) {
		if len(reqdata.DeviceConfig.Key) > int(reqdata.DeviceConfig.AddressNum) {
			rspdata["code"] = 400
			rspdata["message"] = "属性别名数量与线圈数量不匹配！"
		}
	}
	data, err := json.Marshal(rspdata)
	if err != nil {
		log.Println(err.Error())
	}
	fmt.Fprint(w, string(data))
}

func DeleteSubDeviceConfig(w http.ResponseWriter, r *http.Request) {
	r.ParseForm() //解析参数，默认是不会解析的
	log.Println("【收到api请求】path", r.URL.Path)
	log.Println("scheme", r.URL.Scheme)
	log.Println(r.Method)
	if r.Method != "POST" {
		//w.WriteHeader(201)
		return
	}
	var reqdata UpdateSubDeviceConfigStruct
	if err := json.NewDecoder(r.Body).Decode(&reqdata); err != nil {
		r.Body.Close()
		w.WriteHeader(400)
		return
	}
	log.Println("req json: ", reqdata)
	delete(server_map.SubDeviceConfigMap, reqdata.DeviceId) //修改子设备配置只需要修改map中的配置
	var rspdata = make(map[string]interface{})
	w.Header().Set("Content-Type", "application/json")
	rspdata["code"] = 200
	rspdata["message"] = "success"
	data, err := json.Marshal(rspdata)
	if err != nil {
		log.Println(err.Error())
	}
	fmt.Fprint(w, string(data))
}
