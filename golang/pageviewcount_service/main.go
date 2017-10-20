package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/Shopify/sarama"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"
	"github.com/satori/go.uuid"

	"github.com/tgburrin/code_samples/golang/common"
	"github.com/tgburrin/code_samples/golang/dal_postgresql"
	"github.com/tgburrin/code_samples/golang/validation"
)

var iso8601format = "2006-01-02T15:04:05.999999Z07:00"

type processConfiguration struct {
	ApiVersion, KafkaTopic string
	KafkaBrokers           []string
	Settings               map[string]interface{}
	DbCfg                  dal_postgresql.PostgresCfg
	KafkaProducer          sarama.SyncProducer
	DebugOutput            bool
}

var processConfig processConfiguration

var apiVersion = "1"
var debugOutput = false

func sendMessage(kafkaTopic string, kafkaProducer sarama.SyncProducer, msgInput map[string]interface{}) (err error) {
	brv, err := json.Marshal(msgInput)
	if err != nil {
		log.Println("Could not json encode queue message: " + err.Error())
	}

	if debugOutput {
		log.Printf("> sending json to kafka:\n%s\n", brv)
	}

	msg := &sarama.ProducerMessage{Topic: kafkaTopic, Value: sarama.ByteEncoder(brv)}
	if partition, offset, err := kafkaProducer.SendMessage(msg); err != nil {
		return err
	} else {
		if debugOutput {
			log.Printf("> message sent to partition %d at offset %d\n", partition, offset)
		}
	}

	return err
}

// Combine the json input and the url input
func combineInput(w http.ResponseWriter,
	r *http.Request) (inputDoc map[string]interface{}, err error) {

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

func insertClientHandler(w http.ResponseWriter, r *http.Request) {
	_, conerr := dal_postgresql.GetDatabaseHandleFromCfg(&processConfig.DbCfg)
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
		inputDoc, err := combineInput(w, r)
		if err != nil {
			log.Print(err.Error())

			errMsg := make(map[string]interface{})
			common.SetKey(errMsg, "msg", []string{err.Error()})
			common.MakeInvalidInputResponse(w, errMsg)
			return
		}

		fmt.Println(inputDoc)

		return
	default:
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{"Invalid request method"})
		common.MakeInvalidMethodResponse(w, errMsg)
		return
	}
}

func updateClientHandler(w http.ResponseWriter, r *http.Request) {
	_, conerr := dal_postgresql.GetDatabaseHandleFromCfg(&processConfig.DbCfg)
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
	default:
		errMsg := make(map[string]interface{})
		common.SetKey(errMsg, "msg", []string{"Invalid request method"})
		common.MakeInvalidMethodResponse(w, errMsg)
		return
	}
}

func insertContentHandler(w http.ResponseWriter, r *http.Request) {
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

		common.MakeCreatedReponse(w, "/v"+apiVersion+"/content/"+returnRec["content_add"].(string))

		queueMsg := make(map[string]interface{})
		common.SetKey(queueMsg, "type", "content_create")
		common.SetKey(queueMsg, "id", returnRec["content_add"].(string))

		if err = sendMessage(processConfig.KafkaTopic, processConfig.KafkaProducer, queueMsg); err != nil {
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

func updateContentHandler(w http.ResponseWriter, r *http.Request) {
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

		if err = sendMessage(processConfig.KafkaTopic, processConfig.KafkaProducer, queueMsg); err != nil {
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

func processPageviewHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
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
		common.SetKey(queueMsg, "id", inputDoc["id"].(string))
		common.SetKey(queueMsg, "pageview_dt", time.Now().Format(iso8601format))

		if err = sendMessage(processConfig.KafkaTopic, processConfig.KafkaProducer, queueMsg); err != nil {
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

func initialize(processConfig *processConfiguration) {
	var opts struct {
		ConfigFile *string `short:"c" long:"config" description:"Config file"`
	}

	settings := make(map[string]interface{})

	_, err := flags.Parse(&opts)
	if err != nil {
		os.Exit(1)
	}

	if opts.ConfigFile == nil {
		log.Fatal("-c <config file> must be provided\n")
	} else {
		cfgFd, err := os.Open(*opts.ConfigFile)
		if err != nil {
			log.Fatal("Could not parse " + *opts.ConfigFile + ": " + err.Error())
		}

		cfgParser := json.NewDecoder(cfgFd)
		if err = cfgParser.Decode(&settings); err != nil {
			log.Fatal("Could not parse json in " + *opts.ConfigFile + ": " + err.Error())
		}

		err = validation.InitSchema(settings["client_message"].(map[string]interface{}))
		if err != nil {
			panic(err)
		}

		err = validation.InitSchema(settings["content_message"].(map[string]interface{}))
		if err != nil {
			panic(err)
		}
	}

	processConfig.Settings = settings

	if pvc, ok := settings["pageview_connection"].(map[string]interface{}); ok {
		processConfig.KafkaBrokers = common.InterfaceToStringArray(pvc["brokers"])

		if kp, err := sarama.NewSyncProducer(processConfig.KafkaBrokers, nil); err != nil {
			panic(err)
		} else {
			processConfig.KafkaProducer = kp
		}

		if qT, ok := pvc["topic"]; !ok {
			panic("Could not find 'topic' for pageview_connection")
		} else {
			processConfig.KafkaTopic = qT.(string)
		}
	}

	if dbc, ok := settings["content_connection"].(map[string]interface{}); ok {
		processConfig.DbCfg = *dal_postgresql.NewPostgresCfg(dbc)
	}
}

func main() {
	// Setting package global variables
	initialize(&processConfig)

	smux := mux.NewRouter()

	api_r := smux.PathPrefix("/v" + apiVersion).Subrouter()

	client_r := api_r.PathPrefix("/client").Subrouter()
	client_r.HandleFunc("/", insertClientHandler)
	client_r.HandleFunc("/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", updateClientHandler)

	content_r := api_r.PathPrefix("/content").Subrouter()
	content_r.HandleFunc("/", insertContentHandler)
	content_r.HandleFunc("/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", updateContentHandler)

	pageview_r := api_r.PathPrefix("/pageview").Subrouter()
	pageview_r.HandleFunc("/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", processPageviewHandler)

	var loggedRouter http.Handler
	if debugOutput {
		loggedRouter = handlers.LoggingHandler(os.Stdout, smux)
	} else {
		loggedRouter = handlers.LoggingHandler(ioutil.Discard, smux)
	}
	log.Fatal(http.ListenAndServe(":8080", loggedRouter))
}
