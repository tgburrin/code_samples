package apphandlers

import (
	"log"
	"net/http"

	"../common"
	"../dal_postgresql"
	"../validation"
)

func InsertClientHandler(processConfig common.ProcessConfiguration, w http.ResponseWriter, r *http.Request) {
	inputDoc, err := combineInput(w, r)
	if err != nil {
		log.Print(err.Error())

		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{err.Error()})
		common.MakeInvalidInputResponse(w, errMsg)
		return
	}

	conn, conerr := dal_postgresql.GetDatabaseHandleFromCfg(&processConfig.DbCfg)
	if conerr != nil {
		log.Print(conerr.Error())
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{conerr.Error()})
		common.MakeInternalErrorResponse(w, errMsg)
		return
	}

	switch r.Method {
	case "GET":
		return
	case "POST":
		err = validation.ValidateWithSchema(inputDoc, processConfig.Settings["client_message"].(map[string]interface{}), "insert")
		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		cfg := processConfig.Settings["client_action"].(map[string]interface{})["insert"].(map[string]interface{})
		iH, err := dal_postgresql.NewPostgresFunctionExecutor(conn, cfg["function"].(string), cfg["arguments"].([]interface{}))

		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		err = iH.ExecuteProc(inputDoc)
		if err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		returnRec := iH.Record[0].(map[string]interface{})
		common.MakeCreatedReponse(w, "/v"+processConfig.ApiVersion+"/client/"+returnRec["client_id"].(string))

		queueMsg := make(map[string]interface{})
		common.SetKey(queueMsg, "type", "client_create")
		common.SetKey(queueMsg, "id", returnRec["client_id"].(string))

		if err = sendMessage(processConfig, queueMsg); err != nil {
			log.Printf("FAILED to send message: %s\n", err)
		}
		return
	default:
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{"Invalid request method"})
		common.MakeInvalidMethodResponse(w, errMsg)
		return
	}
}

func UpdateClientHandler(processConfig common.ProcessConfiguration, w http.ResponseWriter, r *http.Request) {
	inputDoc, err := combineInput(w, r)
	if err != nil {
		log.Print(err.Error())

		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{err.Error()})
		common.MakeInvalidInputResponse(w, errMsg)
		return
	}

	conn, conerr := dal_postgresql.GetDatabaseHandleFromCfg(&processConfig.DbCfg)
	if conerr != nil {
		log.Print(conerr.Error())
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{conerr.Error()})
		common.MakeInternalErrorResponse(w, errMsg)
		return
	}

	switch r.Method {
	case "GET":
		return
	case "POST":
		err = validation.ValidateWithSchema(inputDoc, processConfig.Settings["client_message"].(map[string]interface{}), "update")
		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		common.SetKey(inputDoc, "replacement", true)

		cfg := processConfig.Settings["client_action"].(map[string]interface{})["update"].(map[string]interface{})
		uH, err := dal_postgresql.NewPostgresFunctionExecutor(conn, cfg["function"].(string), append(cfg["arguments"].([]interface{}), "replacement"))
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		err = uH.ExecuteProc(inputDoc)
		if err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		if uH.NumAffectedLastOp == 1 {
			common.MakeNoContent(w)
		} else {
			common.MakeNotFoundResponse(w, make(map[string]interface{}))
		}

		return

	case "PATCH":
		err = validation.ValidateWithSchema(inputDoc, processConfig.Settings["client_message"].(map[string]interface{}), "update")
		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		cfg := processConfig.Settings["client_action"].(map[string]interface{})["update"].(map[string]interface{})
		uH, err := dal_postgresql.NewPostgresFunctionExecutor(conn, cfg["function"].(string), cfg["arguments"].([]interface{}))
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		err = uH.ExecuteProc(inputDoc)
		if err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		if uH.NumAffectedLastOp == 1 {
			common.MakeNoContent(w)
		} else {
			common.MakeNotFoundResponse(w, make(map[string]interface{}))
		}

		return
	case "DELETE":
		err = validation.ValidateWithSchema(inputDoc, processConfig.Settings["client_message"].(map[string]interface{}), "delete")
		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		cfg := processConfig.Settings["client_action"].(map[string]interface{})["delete"].(map[string]interface{})
		dH, err := dal_postgresql.NewPostgresFunctionExecutor(conn, cfg["function"].(string), cfg["arguments"].([]interface{}))
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		err = dH.ExecuteProc(inputDoc)
		if err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		if dH.NumAffectedLastOp > 0 {
			msg := make(map[string]interface{})
			common.SetKey(msg, "data", dH.Record)
			common.SetKey(msg, "num", len(dH.Record))
			common.SetKey(msg, "next", dH.Record[len(dH.Record)-1])
			common.MakeDataResponse(w, msg)
		} else {
			common.MakeNotFoundResponse(w, make(map[string]interface{}))
		}

		return
	default:
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{"Invalid request method"})
		common.MakeInvalidMethodResponse(w, errMsg)
		return
	}
}
