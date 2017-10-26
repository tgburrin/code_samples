package apphandlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"

	"github.com/gorilla/mux"

	"../common"
	"../dal_postgresql"
	"../validation"
)

func InsertContentHandler(processConfig common.ProcessConfiguration, w http.ResponseWriter, r *http.Request) {
	conn, conerr := dal_postgresql.GetDatabaseHandleFromCfg(&processConfig.DbCfg)
	if conerr != nil {
		log.Print(conerr.Error())
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{conerr.Error()})
		common.MakeInternalErrorResponse(w, errMsg)
		return
	}

	contentTableDetails := make(map[string]interface{})
	dbh, err := dal_postgresql.NewPostgresDataHandler(conn, contentTableDetails)
	if err != nil {
		log.Print(err.Error())
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{err.Error()})
		common.MakeInternalErrorResponse(w, errMsg)
		return
	}

	switch r.Method {
	case "GET":
		args_str := r.FormValue("q")
		if args_str != "" {
			args := make(map[string]interface{})

			if err := json.Unmarshal([]byte(args_str), &args); err != nil {
				errMsg := make(map[string]interface{})
				errorString := "Could not parse json q= argument: " + err.Error()
				log.Print(errorString)
				common.SetKey(errMsg, "msg", []string{errorString})
				common.MakeInvalidInputResponse(w, errMsg)
				return
			}

			if fields, ok := args["fields"]; ok {
				if reflect.TypeOf(fields).Kind() == reflect.Bool && fields.(bool) == true {
					dbh.SetProjection([]string{"*"})
				} else if reflect.TypeOf(fields).Kind() == reflect.Slice {
					l := common.InterfaceToStringArray(fields)
					dbh.SetProjection(l)
				}
			}
		}

		findCriteria := make(map[string]interface{})

		err = validation.ValidateWithSchema(findCriteria,
			processConfig.Settings["content_schema"].(map[string]interface{}),
			"")
		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		dbh.SetFindCriteria(findCriteria)

		err = dbh.FindRecord("return_many", "reverse_sort")
		if err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInternalErrorResponse(w, errMsg)
			return
		}

		if dbh.NumAffectedLastOp > 0 {
			msg := make(map[string]interface{})

			common.SetKey(msg, "data", dbh.Record)
			common.SetKey(msg, "num", len(dbh.Record))
			common.SetKey(msg, "next", dbh.RecordNextIdx)
			common.MakeDataResponse(w, msg)
		} else {
			common.MakeNotFoundResponse(w, make(map[string]interface{}))
		}

		return

	case "POST":
		inputDoc, err := combineInput(w, r)
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		// Create a new identifier for this URL
		//common.SetKey(inputDoc, "id", uuid.NewV1().String())

		// Validate the input vs the legend
		err = validation.ValidateWithSchema(inputDoc,
			processConfig.Settings["content_schema"].(map[string]interface{}),
			"insert")
		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		args := make([]interface{}, 0)
		args = append(args, inputDoc["name"])
		args = append(args, inputDoc["url"])

		err = dbh.ExecuteProc("content_add", args)
		if err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		returnRec := dbh.Record[0].(map[string]interface{})

		common.MakeCreatedReponse(w, "/v"+processConfig.ApiVersion+"/content/"+returnRec["content_add"].(string))

		queueMsg := make(map[string]interface{})
		common.SetKey(queueMsg, "type", "content_create")
		common.SetKey(queueMsg, "id", returnRec["content_add"].(string))

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

func UpdateContentHandler(processConfig common.ProcessConfiguration, w http.ResponseWriter, r *http.Request) {
	conn, conerr := dal_postgresql.GetDatabaseHandleFromCfg(&processConfig.DbCfg)

	if conerr != nil {
		log.Print(conerr.Error())
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{conerr.Error()})
		common.MakeInternalErrorResponse(w, errMsg)
		return
	}

	contentTableDetails := make(map[string]interface{})
	dbh, dberr := dal_postgresql.NewPostgresDataHandler(conn, contentTableDetails)
	if dberr != nil {
		log.Print(dberr.Error())
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{dberr.Error()})
		common.MakeInternalErrorResponse(w, errMsg)
		return
	}

	switch r.Method {
	// Empty response
	case "GET":
		// Need a way of parsing generic query arguments for find 1:N
		/*
		   {
		       "fields":[] or true/false,
		       "query":{ "field1":"equality",
		                 "field2":{">=":"value1","<=":"value2"},
		               },
		       "batchsize": 1000,
		       "order": "asc"/"desc"
		   }
		*/

		args_str := r.FormValue("q")
		if args_str != "" {
			args := make(map[string]interface{})

			if err := json.Unmarshal([]byte(args_str), &args); err != nil {
				errMsg := make(map[string]interface{})
				errorString := "Could not parse json q= argument: " + err.Error()
				log.Print(errorString)
				common.SetKey(errMsg, "msg", []string{errorString})
				common.MakeInvalidInputResponse(w, errMsg)
				return
			}

			if fields, ok := args["fields"]; ok {
				if reflect.TypeOf(fields).Kind() == reflect.Bool && fields.(bool) == true {
					dbh.SetProjection([]string{"*"})
				} else if reflect.TypeOf(fields).Kind() == reflect.Slice {
					l := common.InterfaceToStringArray(fields)
					dbh.SetProjection(l)
				}
			}

			if nextVal, ok := args["next"]; ok {
				// TODO handle next val either here or in the DAL layer.
				// Seems like it should be handled here to form range queries...
				fmt.Printf("Next val is: %v\n", nextVal)
			}
		}

		vars := mux.Vars(r)

		findCriteria := make(map[string]interface{})
		common.SetKey(findCriteria, "id", vars["id"])

		err := validation.ValidateWithSchema(findCriteria,
			processConfig.Settings["content_schema"].(map[string]interface{}),
			"")
		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		dbh.SetFindCriteria(findCriteria)

		err = dbh.FindRecord()
		if err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInternalErrorResponse(w, errMsg)
			return
		}

		if dbh.NumAffectedLastOp > 0 {
			msg := make(map[string]interface{})
			common.SetKey(msg, "data", dbh.Record)
			common.SetKey(msg, "num", len(dbh.Record))
			common.SetKey(msg, "next", dbh.RecordNextIdx)
			common.MakeDataResponse(w, msg)
		} else {
			common.MakeNotFoundResponse(w, make(map[string]interface{}))
		}

		return

	case "PATCH":
		findCriteria := make(map[string]interface{})

		inputDoc, err := combineInput(w, r)
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		// Validate the input vs the legend
		err = validation.ValidateWithSchema(inputDoc,
			processConfig.Settings["content_schema"].(map[string]interface{}),
			"update")
		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		common.SetKey(findCriteria, "id", inputDoc["id"])
		delete(inputDoc, "id")

		dbh.SetFindCriteria(findCriteria)
		if err = dbh.UpdateRecord(inputDoc); err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInternalErrorResponse(w, errMsg)
			return
		}

		if dbh.NumAffectedLastOp > 0 {
			common.MakeNoContent(w)
		} else {
			common.MakeNotFoundResponse(w, make(map[string]interface{}))
		}

		return

	case "POST":
		findCriteria := make(map[string]interface{})

		inputDoc, err := combineInput(w, r)
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		// Validate the input vs the legend
		err = validation.ValidateWithSchema(inputDoc,
			processConfig.Settings["content_schema"].(map[string]interface{}),
			"replace")
		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		common.SetKey(findCriteria, "id", inputDoc["id"])
		delete(inputDoc, "id")

		dbh.SetFindCriteria(findCriteria)
		if err = dbh.UpdateRecord(inputDoc); err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInternalErrorResponse(w, errMsg)
			return
		}

		if dbh.NumAffectedLastOp > 0 {
			common.MakeNoContent(w)
		} else {
			common.MakeNotFoundResponse(w, make(map[string]interface{}))
		}

		return

	case "DELETE":
		findCriteria := make(map[string]interface{})

		fmt.Println("About to combineInput")
		inputDoc, err := combineInput(w, r)
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		//fmt.Printf("inputDoc Addr: %p\n", inputDoc)

		// Validate the input vs the legend
		err = validation.ValidateWithSchema(inputDoc,
			processConfig.Settings["content_schema"].(map[string]interface{}),
			"delete")

		if err != nil {
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		common.SetKey(findCriteria, "id", inputDoc["id"])
		delete(inputDoc, "id")

		dbh.SetFindCriteria(findCriteria)

		if err = dbh.DeleteRecord(); err != nil {
			log.Print(err.Error())
			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInternalErrorResponse(w, errMsg)
			return
		}

		if dbh.NumAffectedLastOp > 0 {
			common.MakeNoContent(w)
		} else {
			common.MakeNotFoundResponse(w, make(map[string]interface{}))
		}

		queueMsg := make(map[string]interface{})
		common.SetKey(queueMsg, "type", "content_delete")
		common.SetKey(queueMsg, "id", findCriteria["id"].(string))

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
