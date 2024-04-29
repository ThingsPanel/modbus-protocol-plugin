package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"

	globaldata "github.com/ThingsPanel/modbus-protocol-plugin/global_data"
	httpclient "github.com/ThingsPanel/modbus-protocol-plugin/http_client"
	"github.com/ThingsPanel/modbus-protocol-plugin/services"
	service "github.com/ThingsPanel/modbus-protocol-plugin/services"
	tpprotocolsdkgo "github.com/ThingsPanel/tp-protocol-sdk-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var HttpClient *tpprotocolsdkgo.Client

func Init() {
	go start()
}

func start() {
	var handler tpprotocolsdkgo.Handler = tpprotocolsdkgo.Handler{
		// OnCreateDevice: OnCreateDevice,
		// OnUpdateDevice: OnUpdateDevice,
		// OnDeleteDevice: OnDeleteDevice,
		OnGetForm: OnGetForm,
	}
	addr := viper.GetString("http_server.address")
	logrus.Info("http服务启动：", addr)
	err := handler.ListenAndServe(addr)
	if err != nil {
		logrus.Info("ListenAndServe() failed, err: ", err)
		return
	}
}

// OnCreateDevice 创建设备
func OnCreateDevice(w http.ResponseWriter, r *http.Request) {
	logrus.Info("OnCreateDevice")
	r.ParseForm() //解析参数，默认是不会解析的
	logrus.Info("【收到api请求】path", r.URL.Path)
	logrus.Info("scheme", r.URL.Scheme)
	// 读取客户端发送的数据
	var reqDataMap = make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&reqDataMap); err != nil {
		r.Body.Close()
		w.WriteHeader(400)
		return
	}

	gateWayID := reqDataMap["ParentId"].(string)
	err := updateGatewayConfig(gateWayID)
	if err != nil {
		logrus.Info(err.Error())
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

// OnUpdateDevice 更新设备
func OnUpdateDevice(w http.ResponseWriter, r *http.Request) {
	logrus.Info("OnUpdateDevice")
	r.ParseForm() //解析参数，默认是不会解析的
	logrus.Info("【收到api请求】path", r.URL.Path)
	logrus.Info("scheme", r.URL.Scheme)
	// 读取客户端发送的数据
	var reqDataMap = make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&reqDataMap); err != nil {
		r.Body.Close()
		w.WriteHeader(400)
		return
	}

	gateWayID := reqDataMap["ParentId"].(string)
	err := updateGatewayConfig(gateWayID)
	if err != nil {
		logrus.Info(err.Error())
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

// OnDeleteDevice 删除设备
func OnDeleteDevice(w http.ResponseWriter, r *http.Request) {
	logrus.Info("OnDeleteDevice")
	r.ParseForm() //解析参数，默认是不会解析的
	logrus.Info("【收到api请求】path", r.URL.Path)
	logrus.Info("scheme", r.URL.Scheme)
	// 读取客户端发送的数据
	var reqDataMap = make(map[string]interface{})
	if err := json.NewDecoder(r.Body).Decode(&reqDataMap); err != nil {
		r.Body.Close()
		w.WriteHeader(400)
		return
	}
	deviceType := reqDataMap["DeviceType"].(string)
	// 子设备
	if deviceType == "3" {
		gateWayID := reqDataMap["ParentId"].(string)
		err := updateGatewayConfig(gateWayID)
		if err != nil {
			logrus.Info(err.Error())
			r.Body.Close()
			w.WriteHeader(400)
			return
		}
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

	device_type := r.URL.Query()["device_type"][0]
	form_type := r.URL.Query()["form_type"][0]
	protocol_type := r.URL.Query()["protocol_type"][0]
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
			RspSuccess(w, readFormConfigByPath("./form_config.json"))
		} else {
			RspSuccess(w, nil)
		}
	case "VCR":
		if device_type == "2" {
			// 网关凭证表单
			RspSuccess(w, readFormConfigByPath("./form_voucher.json"))
		} else {
			RspSuccess(w, nil)
		}
	case "VCRT":
		if device_type == "2" {
			// 网关凭证类型表单

			RspSuccess(w, readFormConfigByPath("./form_voucher_type.json"))
		} else {
			RspSuccess(w, nil)
		}
	default:
		RspError(w, errors.New("not support form type: "+form_type))
	}
}

// 更新配置
func updateGatewayConfig(gateWayID string) error {
	// 获取网关配置
	gatewayConfig, err := httpclient.GetDeviceConfig("", gateWayID)
	if err != nil {
		return err
	}
	logrus.Info("网关配置：", gatewayConfig.Data)
	// 获取连接
	conn, ok := globaldata.DeviceConnectionMap.Load(gatewayConfig.Data.Voucher)
	if ok {
		c := *conn.(*net.Conn)
		// 如果本身是关闭的也无所谓，它会在读和写的时候返回错误
		service.CloseConnection(c, gatewayConfig.Data.Voucher)
	} else {
		return errors.New("Connection not found for token:" + gatewayConfig.Data.Voucher)
	}
	// 更换配置
	globaldata.GateWayConfigMap.Store(gatewayConfig.Data.Voucher, &gatewayConfig.Data)
	// 将设备连接存入全局变量
	services.HandleConn(gatewayConfig.Data.Voucher, gatewayConfig.Data.ID) // 处理连接
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
