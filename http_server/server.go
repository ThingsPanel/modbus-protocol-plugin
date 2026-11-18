package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	"github.com/ThingsPanel/modbus-protocol-plugin/mqtt"
	service "github.com/ThingsPanel/modbus-protocol-plugin/services"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func Serve() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/form/config", OnGetForm)
	mux.HandleFunc("POST /api/v1/device/disconnect", OnDisconnectDevice)
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		if !mqtt.Ready() {
			http.Error(w, "MQTT unavailable", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	server := &http.Server{
		Addr: viper.GetString("http_server.address"), Handler: mux, ReadHeaderTimeout: 5 * time.Second,
	}
	return server.ListenAndServe()
}

// OnDisconnectDevice 断开设备
func OnDisconnectDevice(w http.ResponseWriter, r *http.Request) {
	logrus.Info("OnDisconnectDevice")
	r.ParseForm() //解析参数，默认是不会解析的
	logrus.Info("【收到api请求】path", r.URL.Path)
	// 读取客户端发送的数据
	var reqDataMap = make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&reqDataMap); err != nil {
		r.Body.Close()
		w.WriteHeader(400)
		return
	}
	logrus.Info("reqDataMap:", reqDataMap)
	// 判断reqDataMap是否有key为device_id的值
	if _, ok := reqDataMap["device_id"]; !ok {
		RspError(w, errors.New("device_id not found"))
		return
	}
	// 判断reqDataMap["device_id"]的数据类型是否为string
	if _, ok := reqDataMap["device_id"].(string); !ok {
		RspError(w, errors.New("device_id type error"))
		return
	}
	deviceID := reqDataMap["device_id"].(string)
	err := disconnectDevice(deviceID)
	if err != nil {
		logrus.Error(err.Error())
		r.Body.Close()
		w.WriteHeader(400)
		return
	}

	// 返回成功
	var rspdata = make(map[string]interface{})
	w.Header().Set("Content-Type", "application/json")
	rspdata["code"] = 200
	rspdata["message"] = "success"
	data, err := json.Marshal(rspdata)
	if err != nil {
		logrus.Info(err.Error())
	}
	fmt.Fprint(w, string(data))
}

// OnGetForm 获取协议插件的json表单
func OnGetForm(w http.ResponseWriter, r *http.Request) {
	logrus.Info("OnGetForm")
	r.ParseForm() //解析参数，默认是不会解析的
	logrus.Info("【收到api请求】path", r.URL.Path)
	logrus.Info("query", r.URL.Query())

	device_type := r.URL.Query().Get("device_type")
	form_type := r.URL.Query().Get("form_type")
	protocol_type := r.URL.Query().Get("protocol_type")
	formDir := viper.GetString("http_server.form_dir")
	// 如果请求参数protocol_type不等于MODBUS_RTU或MODBUS_TCP，返回空
	if protocol_type != "MODBUS_RTU" && protocol_type != "MODBUS_TCP" {
		RspError(w, errors.New("not support protocol type"))
		return
	}
	//CFG配置表单 VCR凭证表单 VCRT凭证类型表单
	switch form_type {
	case "CFG":
		if device_type == "3" {
			// 子设备配置表单
			RspSuccess(w, readFormConfigByPath(filepath.Join(formDir, "form_config.json")))
		} else {
			RspSuccess(w, nil)
		}
	case "VCR":
		if device_type == "2" {
			// 网关凭证表单
			RspSuccess(w, readFormConfigByPath(filepath.Join(formDir, "form_voucher.json")))
		} else {
			RspSuccess(w, nil)
		}
	case "VCRT":
		if device_type == "2" {
			// 网关凭证类型表单

			RspSuccess(w, readFormConfigByPath(filepath.Join(formDir, "form_voucher_type.json")))
		} else {
			RspSuccess(w, nil)
		}
	default:
		RspError(w, errors.New("not support form type: "+form_type))
	}
}

// 断开设备连接
func disconnectDevice(deviceID string) error {
	// 获取连接
	conn, ok := globaldata.DeviceConnectionMap.Load(deviceID)
	if ok {
		c := *conn.(*net.Conn)
		// 如果本身是关闭的也无所谓，它会在读和写的时候返回错误
		service.CloseConnection(c, deviceID)
	} else {
		return errors.New("Connection not found for deviceID:" + deviceID)
	}
	return nil
}

// ./form_config.json
func readFormConfigByPath(path string) interface{} {
	filePtr, err := os.Open(path)
	if err != nil {
		logrus.Info("文件打开失败...", err.Error())
		return nil
	}
	defer filePtr.Close()
	var info interface{}
	// 创建json解码器
	decoder := json.NewDecoder(filePtr)
	err = decoder.Decode(&info)
	if err != nil {
		logrus.Info("解码失败", err.Error())
		return info
	} else {
		logrus.Info("读取文件[form_config.json]成功...")
		return info
	}
}

// OnCreateDevice 创建设备
// func OnCreateDevice(w http.ResponseWriter, r *http.Request) {
// 	logrus.Info("OnCreateDevice")
// 	r.ParseForm() //解析参数，默认是不会解析的
// 	logrus.Info("【收到api请求】path", r.URL.Path)
// 	logrus.Info("scheme", r.URL.Scheme)
// 	// 读取客户端发送的数据
// 	var reqDataMap = make(map[string]interface{})
// 	if err := json.NewDecoder(r.Body).Decode(&reqDataMap); err != nil {
// 		r.Body.Close()
// 		w.WriteHeader(400)
// 		return
// 	}

// 	gateWayID := reqDataMap["ParentId"].(string)
// 	err := updateGatewayConfig(gateWayID)
// 	if err != nil {
// 		logrus.Info(err.Error())
// 		r.Body.Close()
// 		w.WriteHeader(400)
// 		return
// 	}
// 	// 返回成功
// 	var rspdata = make(map[string]interface{})
// 	w.Header().Set("Content-Type", "application/json")
// 	rspdata["code"] = 200
// 	rspdata["message"] = "success"
// 	data, err := json.Marshal(rspdata)
// 	if err != nil {
// 		logrus.Info(err.Error())
// 	}
// 	fmt.Fprint(w, string(data))
// }

// OnUpdateDevice 更新设备
// func OnUpdateDevice(w http.ResponseWriter, r *http.Request) {
// 	logrus.Info("OnUpdateDevice")
// 	r.ParseForm() //解析参数，默认是不会解析的
// 	logrus.Info("【收到api请求】path", r.URL.Path)
// 	logrus.Info("scheme", r.URL.Scheme)
// 	// 读取客户端发送的数据
// 	var reqDataMap = make(map[string]interface{})
// 	if err := json.NewDecoder(r.Body).Decode(&reqDataMap); err != nil {
// 		r.Body.Close()
// 		w.WriteHeader(400)
// 		return
// 	}

// 	gateWayID := reqDataMap["ParentId"].(string)
// 	err := updateGatewayConfig(gateWayID)
// 	if err != nil {
// 		logrus.Info(err.Error())
// 		r.Body.Close()
// 		w.WriteHeader(400)
// 		return
// 	}
// 	// 返回成功
// 	var rspdata = make(map[string]interface{})
// 	w.Header().Set("Content-Type", "application/json")
// 	rspdata["code"] = 200
// 	rspdata["message"] = "success"
// 	data, err := json.Marshal(rspdata)
// 	if err != nil {
// 		logrus.Info(err.Error())
// 	}
// 	fmt.Fprint(w, string(data))
// }

// OnDeleteDevice 删除设备
// func OnDeleteDevice(w http.ResponseWriter, r *http.Request) {
// 	logrus.Info("OnDeleteDevice")
// 	r.ParseForm() //解析参数，默认是不会解析的
// 	logrus.Info("【收到api请求】path", r.URL.Path)
// 	logrus.Info("scheme", r.URL.Scheme)
// 	// 读取客户端发送的数据
// 	var reqDataMap = make(map[string]interface{})
// 	if err := json.NewDecoder(r.Body).Decode(&reqDataMap); err != nil {
// 		r.Body.Close()
// 		w.WriteHeader(400)
// 		return
// 	}
// 	deviceType := reqDataMap["DeviceType"].(string)
// 	// 子设备
// 	if deviceType == "3" {
// 		gateWayID := reqDataMap["ParentId"].(string)
// 		err := updateGatewayConfig(gateWayID)
// 		if err != nil {
// 			logrus.Info(err.Error())
// 			r.Body.Close()
// 			w.WriteHeader(400)
// 			return
// 		}
// 	}
// 	// 返回成功
// 	var rspdata = make(map[string]interface{})
// 	w.Header().Set("Content-Type", "application/json")
// 	rspdata["code"] = 200
// 	rspdata["message"] = "success"
// 	data, err := json.Marshal(rspdata)
// 	if err != nil {
// 		logrus.Info(err.Error())
// 	}
// 	fmt.Fprint(w, string(data))
// }

// 更新配置
// func updateGatewayConfig(gateWayID string) error {
// 	// 获取网关配置
// 	gatewayConfig, err := httpclient.GetDeviceConfig("", gateWayID)
// 	if err != nil {
// 		return err
// 	}
// 	logrus.Info("网关配置：", gatewayConfig.Data)
// 	// 获取连接
// 	conn, ok := globaldata.DeviceConnectionMap.Load(gatewayConfig.Data.Voucher)
// 	if ok {
// 		c := *conn.(*net.Conn)
// 		// 如果本身是关闭的也无所谓，它会在读和写的时候返回错误
// 		service.CloseConnection(c, gatewayConfig.Data.Voucher)
// 	} else {
// 		return errors.New("Connection not found for token:" + gatewayConfig.Data.Voucher)
// 	}
// 	// 更换配置
// 	globaldata.GateWayConfigMap.Store(gatewayConfig.Data.Voucher, &gatewayConfig.Data)
// 	// 将设备连接存入全局变量
// 	services.HandleConn(gatewayConfig.Data.Voucher, gatewayConfig.Data.ID) // 处理连接
// 	return nil
// }
