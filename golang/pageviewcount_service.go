package main 

import (
    "os"
    "fmt"
    "log"
    "reflect"
    "errors"
    "encoding/json"
    "net/http"
    "io/ioutil"
    "time"

    "github.com/Shopify/sarama"
    "github.com/jessevdk/go-flags"
    "github.com/satori/go.uuid"
    "github.com/gorilla/mux"
    "github.com/gorilla/handlers"

    "github.com/tgburrin/rest_utilities/common"
    "github.com/tgburrin/rest_utilities/validation"
    "github.com/tgburrin/rest_utilities/dal_postgresql"
)

var iso8601format = "2006-01-02T15:04:05.999999Z07:00"

var settings map[string]interface{}
var contentTableDetails map[string]interface{}
var batchSize = 10
var apiVersion = "1"

var queueProducer sarama.SyncProducer
var queueTopic string

var debugOutput = false

func sendMessage ( msgInput map[string]interface{} ) ( err error ) {
    brv, err := json.Marshal(msgInput)
    if err != nil {
        log.Println("Could not json encode queue message: "+err.Error())
    }

    if debugOutput {
        log.Printf("> sending json to kafka:\n%s\n", brv)
    }
    
    msg := &sarama.ProducerMessage{Topic: queueTopic, Value: sarama.ByteEncoder(brv)}
    if partition, offset, err := queueProducer.SendMessage(msg); err != nil {
        return err
    } else {
        if debugOutput {
            log.Printf("> message sent to partition %d at offset %d\n", partition, offset)
        }
    }

    return err
}

// Combine the json input and the url input
func combineInput ( w http.ResponseWriter,
                    r *http.Request ) ( inputDoc map[string]interface{}, err error ) {

    vars := mux.Vars(r)
    inputDoc = make(map[string]interface{})

    // Combine the json input and the url input
    if body, err := ioutil.ReadAll(r.Body); err != nil {
        return inputDoc, errors.New("Error reading http body: "+err.Error())
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
                return inputDoc, errors.New("Could not add/append arguments to input: "+err.Error())
            }
        }
    }

    return inputDoc, err
}

func insertContentHandler (w http.ResponseWriter, r *http.Request) {
    conn, err := dal_postgresql.GetDatabaseHandle(
                    settings["content_connection"].(map[string]interface{}))

    if err != nil {
        log.Print(err.Error())
        errMsg := make(map[string]interface{})
        common.SetKey(errMsg, "msg", []string{err.Error()})
        common.MakeInternalErrorResponse(w, errMsg)
        return
    }

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
                    errorString := "Could not parse json q= argument: "+err.Error()
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
                                                settings["content_schema"].(map[string]interface{}),
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
                                                settings["content_schema"].(map[string]interface{}),
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

            if err = sendMessage(queueMsg); err != nil {
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

func updateContentHandler (w http.ResponseWriter, r *http.Request) {
    conn, conerr := dal_postgresql.GetDatabaseHandle(
                    settings["content_connection"].(map[string]interface{}))

    if conerr != nil {
        log.Print(conerr.Error())
        errMsg := make(map[string]interface{})
        common.SetKey(errMsg, "msg", []string{conerr.Error()})
        common.MakeInternalErrorResponse(w, errMsg)
        return
    }

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
                    errorString := "Could not parse json q= argument: "+err.Error()
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
                                                settings["content_schema"].(map[string]interface{}),
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
                                                settings["content_schema"].(map[string]interface{}),
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
                                                settings["content_schema"].(map[string]interface{}),
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
                                                settings["content_schema"].(map[string]interface{}),
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

            if err = sendMessage(queueMsg); err != nil {
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

func processPageviewHandler ( w http.ResponseWriter, r *http.Request ) {
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

            if err = sendMessage(queueMsg); err != nil {
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

func initialize () {
    var opts struct {
        ConfigFile *string `short:"c" long:"config" description:"Config file"`
    }

    settings = make(map[string]interface{})

    _, err := flags.Parse(&opts)
    if err != nil {
        os.Exit(1)
    }

    if opts.ConfigFile == nil {
        log.Fatal("-c <config file> must be provided\n")
    } else {
        cfgFd, err := os.Open(*opts.ConfigFile)
        if err != nil {
            log.Fatal("Could not parse "+*opts.ConfigFile+": "+err.Error())
        }

        cfgParser := json.NewDecoder(cfgFd)
        if err = cfgParser.Decode(&settings); err != nil {
            log.Fatal("Could not parse json in "+*opts.ConfigFile+": "+err.Error())
        }
        err = validation.InitSchema(settings["content_schema"].(map[string]interface{}))
        if err != nil {
            panic(err)
        }
    }

    if pvc, ok := settings["pageview_connection"].(map[string]interface{}); ok {
        brokers := common.InterfaceToStringArray(pvc["brokers"])

        if queueProducer, err = sarama.NewSyncProducer(brokers, nil); err != nil {
            panic(err)
        }

        if qT, ok := pvc["topic"]; !ok {
            panic("Could not find 'topic' for pageview_connection")
        } else {
            queueTopic = qT.(string)
        }
    }

    contentTableDetails = make(map[string]interface{})
    common.SetKey(contentTableDetails, "table_name", "content")
    common.SetKey(contentTableDetails, "pk", []string{"id"})
}

func main () {
    initialize()

    smux := mux.NewRouter()
    smux.HandleFunc("/v"+apiVersion+"/content", insertContentHandler)
    smux.HandleFunc("/v"+apiVersion+
                    "/content/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}",
                     updateContentHandler)
    smux.HandleFunc("/v"+apiVersion+
                    "/pageview/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}",
                     processPageviewHandler)

    var loggedRouter http.Handler
    if debugOutput {
        loggedRouter = handlers.LoggingHandler(os.Stdout, smux)
    } else {
        loggedRouter = handlers.LoggingHandler(ioutil.Discard, smux)
    }
    log.Fatal(http.ListenAndServe(":8080", loggedRouter))
}
