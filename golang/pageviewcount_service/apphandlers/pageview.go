package apphandlers

import (
	"log"
	"net/http"
	"time"

	"github.com/satori/go.uuid"

	"../common"
)

func ProcessPageviewHandler(processConfig common.ProcessConfiguration, w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		inputDoc, err := combineInput(w, r)
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		_, err = uuid.FromString(inputDoc["id"].(string))
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		queueMsg := make(map[string]interface{})
		common.SetKey(queueMsg, "type", "content_pageview")
		common.SetKey(queueMsg, "content_id", inputDoc["id"].(string))
		common.SetKey(queueMsg, "pageview_dt", time.Now().Format(processConfig.TimeFormat))

		if err = sendMessage(processConfig, queueMsg); err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInternalErrorResponse(w, errMsg)
			return
		} else {
			common.MakeNoContent(w)
		}

		return

	default:
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{"Invalid request method"})
		common.MakeInvalidMethodResponse(w, errMsg)
		return
	}
}
