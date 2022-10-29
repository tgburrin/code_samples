package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/Shopify/sarama"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/jessevdk/go-flags"

	"./apphandlers"
	"./common"
	"./dal_postgresql"
	"./validation"
)

type handlerFunction func(common.ProcessConfiguration, http.ResponseWriter, *http.Request)

var processConfig = common.ProcessConfiguration{ApiVersion: "1", DebugOutput: false, TimeFormat: "2006-01-02T15:04:05.999999Z07:00"}

func HandlerWrapper(r *mux.Router, path string, fn handlerFunction) {
	r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		fn(processConfig, w, r)
	})
}

func init() {
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
	smux := mux.NewRouter()

	api_r := smux.PathPrefix("/v" + processConfig.ApiVersion).Subrouter()

	client_r := api_r.PathPrefix("/client").Subrouter()
	HandlerWrapper(client_r, "/", apphandlers.InsertClientHandler)
	HandlerWrapper(client_r, "/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", apphandlers.UpdateClientHandler)

	content_r := api_r.PathPrefix("/content").Subrouter()
	HandlerWrapper(content_r, "/", apphandlers.InsertContentHandler)
	HandlerWrapper(content_r, "/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", apphandlers.UpdateContentHandler)

	pageview_r := api_r.PathPrefix("/pageview").Subrouter()
	HandlerWrapper(pageview_r, "/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", apphandlers.ProcessPageviewHandler)

	var loggedRouter http.Handler
	if processConfig.DebugOutput {
		loggedRouter = handlers.LoggingHandler(os.Stdout, smux)
	} else {
		loggedRouter = handlers.LoggingHandler(ioutil.Discard, smux)
	}
	log.Fatal(http.ListenAndServe(":8080", loggedRouter))
}
