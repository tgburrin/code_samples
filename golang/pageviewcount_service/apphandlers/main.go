package apphandlers

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/Shopify/sarama"
	"github.com/gorilla/mux"

	"../common"
)

func sendMessage(processConfig common.ProcessConfiguration, msgInput map[string]interface{}) (err error) {
	brv, err := json.Marshal(msgInput)
	if err != nil {
		log.Println("Could not json encode queue message: " + err.Error())
	}

	if processConfig.DebugOutput {
		log.Printf("> sending json to kafka:\n%s\n", brv)
	}

	msg := &sarama.ProducerMessage{Topic: processConfig.KafkaTopic, Value: sarama.ByteEncoder(brv)}
	if partition, offset, err := processConfig.KafkaProducer.SendMessage(msg); err != nil {
		return err
	} else {
		if processConfig.DebugOutput {
			log.Printf("> message sent to partition %d at offset %d\n", partition, offset)
		}
	}

	return err
}

// Combine the json input and the url input
func combineInput(w http.ResponseWriter, r *http.Request) (inputDoc map[string]interface{}, err error) {

	vars := mux.Vars(r)
	inputDoc = make(map[string]interface{})

	// Combine the json input and the url input
	if body, err := ioutil.ReadAll(r.Body); err != nil {
		return inputDoc, errors.New("Error reading http body: " + err.Error())
	} else {
		if len(body) > 0 {
			if err = json.Unmarshal(body, &inputDoc); err != nil {
				return inputDoc, err
			}
		}
	}

	if len(vars) > 0 {
		for argn, argv := range vars {
			if ok, err := common.SetKey(inputDoc, argn, argv); !ok {
				return inputDoc, errors.New("Could not add/append arguments to input: " + err.Error())
			}
		}
	}

	return inputDoc, err
}
