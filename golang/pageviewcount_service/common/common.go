package common

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"

	"github.com/Shopify/sarama"

	"../dal_postgresql"
)

type ProcessConfiguration struct {
	ApiVersion, KafkaTopic string
	KafkaBrokers           []string
	Settings               map[string]interface{}
	DbCfg                  dal_postgresql.PostgresCfg
	KafkaProducer          sarama.SyncProducer
	DebugOutput            bool
	TimeFormat             string
}

type Dict map[string]interface{}

func (d *Dict) SetKey(key string, value interface{}) (rv bool, err error) {
	dict := *d
	rv = true

	if dict == nil {
		rv = false
		err = errors.New("Uninitialized map passed")
		return
	}

	if _, ok := dict[key]; !ok {
		dict[key] = make([]interface{}, 0)
	}
	dict[key] = value

	return
}

func NewDict() (d Dict) {
	d = make(map[string]interface{})
	return
}

// A method for updating map keys.  It shoudl be noted that this is not thread safe.
func SetKey(dict map[string]interface{}, key string, value interface{}) (rv bool, err error) {
	rv = true

	if dict == nil {
		rv = false
		err = errors.New("Uninitialized map passed")
		return
	}

	if _, ok := dict[key]; !ok {
		dict[key] = make([]interface{}, 0)
	}
	dict[key] = value

	return
}

func InterfaceToStringArray(input_if interface{}) (output []string) {
	input := input_if.([]interface{})
	output = make([]string, len(input))
	for i, v := range input {
		output[i] = v.(string)
		//output = append(output, v.(string))
	}
	return
}

func formatResponse(input map[string]interface{}) (rv map[string]interface{}) {
	rv = make(map[string]interface{})

	if val, ok := input["msg"]; ok && reflect.TypeOf(val).Elem().Kind() == reflect.String {
		SetKey(rv, "errormsg", val)
	}

	if val, ok := input["data"]; ok {
		SetKey(rv, "data", val)
	}

	if val, ok := input["num"]; ok && reflect.TypeOf(val).Kind() == reflect.Int {
		SetKey(rv, "num", val)
	}

	if val, ok := input["next"]; ok && reflect.TypeOf(val).Kind() == reflect.Map {
		SetKey(rv, "next", val)
	}

	return
}

func formatJsonResponse(input map[string]interface{}) (rv string, err error) {
	brv, err := json.Marshal(formatResponse(input))
	rv = string(brv)
	return
}

func MakeCreatedReponse(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
	w.WriteHeader(http.StatusCreated)
}

func MakeNoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func MakeDataResponse(w http.ResponseWriter, input map[string]interface{}) (err error) {
	if jsonMsg, err := formatJsonResponse(input); err != nil {
		return err
	} else {
		io.WriteString(w, jsonMsg+"\n")
	}
	return
}

func MakeNotFoundResponse(w http.ResponseWriter, input map[string]interface{}) (err error) {
	if jsonMsg, err := formatJsonResponse(input); err != nil {
		return err
	} else {
		w.WriteHeader(http.StatusNotFound)
		io.WriteString(w, jsonMsg+"\n")
	}

	return
}

func MakeInvalidInputResponse(w http.ResponseWriter, input map[string]interface{}) (err error) {
	if jsonMsg, err := formatJsonResponse(input); err != nil {
		return err
	} else {
		http.Error(w, jsonMsg, 400)
	}
	return
}

func MakeInvalidMethodResponse(w http.ResponseWriter, input map[string]interface{}) (err error) {
	jsonMsg, err := formatJsonResponse(input)
	if err != nil {
		return err
	}

	http.Error(w, jsonMsg, 405)
	return err
}

func MakeInternalErrorResponse(w http.ResponseWriter, input map[string]interface{}) (err error) {
	jsonMsg, err := formatJsonResponse(input)
	if err != nil {
		return err
	}

	http.Error(w, jsonMsg, 500)
	return err
}

func MakeNotImplementedResponse(w http.ResponseWriter, input map[string]interface{}) (err error) {
	jsonMsg, err := formatJsonResponse(input)
	if err != nil {
		return err
	}

	http.Error(w, jsonMsg, 501)
	return err
}

func MakeUnavailableResponse(w http.ResponseWriter, input map[string]interface{}) (err error) {
	jsonMsg, err := formatJsonResponse(input)
	if err != nil {
		return err
	}

	http.Error(w, jsonMsg, 503)
	return err
}
