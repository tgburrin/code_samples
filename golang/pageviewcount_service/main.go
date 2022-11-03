package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

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
		fmt.Println("Reading configuration from " + *opts.ConfigFile)
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
	fmt.Println("Configuration parsed, loading values")

	if pvc, ok := settings["pageview_connection"].(map[string]interface{}); ok {
		fmt.Println("Adding kafka pageview_connection details")
		processConfig.KafkaBrokers = common.InterfaceToStringArray(pvc["brokers"])

		if kp, err := sarama.NewSyncProducer(processConfig.KafkaBrokers, nil); err != nil {
			panic(err)
		} else {
			processConfig.KafkaProducer = kp
			fmt.Println("Kafka producer sending to " + strings.Join(processConfig.KafkaBrokers, ", "))
		}

		if qT, ok := pvc["topic"]; !ok {
			panic("Could not find 'topic' for pageview_connection")
		} else {
			processConfig.KafkaTopic = qT.(string)
			fmt.Println("Kafka topic set to " + processConfig.KafkaTopic)
		}
	}

	if dbc, ok := settings["content_connection"].(map[string]interface{}); ok {
		fmt.Println("Adding postgres content details")
		processConfig.DbCfg = *dal_postgresql.NewPostgresCfg(dbc)
	}

	if dbg, ok := settings["logging_level"]; ok {
		if dbg.(string) == "debug" {
			fmt.Println("Enabled debug logging")
			processConfig.DebugOutput = true
		}
	}
}

func main() {
	smux := mux.NewRouter()

	api_r := smux.PathPrefix("/v" + processConfig.ApiVersion).Subrouter()

	client_r := api_r.PathPrefix("/client").Subrouter()
	HandlerWrapper(client_r, "/maintain", apphandlers.InsertClientHandler)
	HandlerWrapper(client_r, "/maintain/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", apphandlers.UpdateClientHandler)
	HandlerWrapper(client_r, "/read/find/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", apphandlers.FindOneClientHandler)
	HandlerWrapper(client_r, "/read/list", apphandlers.FindManyClientHandler)

	content_r := api_r.PathPrefix("/content").Subrouter()
	HandlerWrapper(content_r, "/maintain", apphandlers.InsertContentHandler)
	HandlerWrapper(content_r, "/maintain/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", apphandlers.UpdateContentHandler)
	//HandlerWrapper(content_r, "/read/find/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", apphandlers.FindOneContentHandler)
	HandlerWrapper(content_r, "/read/list", apphandlers.FindManyContentHandler)

	pageview_r := api_r.PathPrefix("/pageview").Subrouter()
	HandlerWrapper(pageview_r, "/content/{id:[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}}", apphandlers.ProcessPageviewHandler)

	var loggedRouter http.Handler
	if processConfig.DebugOutput {
		loggedRouter = handlers.LoggingHandler(os.Stdout, smux)
	} else {
		loggedRouter = handlers.LoggingHandler(ioutil.Discard, smux)
	}
	log.Fatal(http.ListenAndServe(":8080", loggedRouter))
}
