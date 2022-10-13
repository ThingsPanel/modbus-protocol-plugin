package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
)

func PostJson(url string, req interface{}) ([]byte, error) {
	body, _ := json.Marshal(req)
	log.Println(string(body))
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		log.Println("Response: ", string(body))
		return body, err
	} else {
		return nil, errors.New("Get failed with error: " + resp.Status)
	}
}
