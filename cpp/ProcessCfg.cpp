/*
 * ProcessCfg.cpp
 *
 *  Created on: Oct 10, 2017
 *      Author: tgburrin
 */

#include "ProcessCfg.h"

ProcessCfg::ProcessCfg(string cfgFile) {
    batchCommitSize = 1000;
    kafkaDebug = false;

    _ParseConfig(cfgFile);
}

ProcessCfg::~ProcessCfg() {
}

void ProcessCfg::_ParseConfig (string config_file) {
	Json::Value doc;
    Json::Reader jr;

    Json::Value v;

    ifstream infile(config_file.c_str());
    jr.parse(infile, doc, false);
    infile.close();

    if ( !jr.good() )
    	throw ApplicationException("Could not parse config file");

    if ( !doc["batch_commit_size"].isNull() )
        SetBatchSize(doc["batch_commit_size"].asInt());

    // Kafka details
    if ( doc["pageview_connection"].isNull() )
    	throw ApplicationException("The 'pageview_connection' section of the config must be specified");

	v = doc["pageview_connection"]["brokers"];

	if ( v.isNull() )
		throw ApplicationException("The 'brokers' section of pageview_connection must be specified");

	for(uint i=0; i < v.size(); i++) {
		string val = v[i].asString();
		if ( !val.empty() )
			brokers.push_back(val);
	}

	if ( brokers.size() == 0 )
		throw ApplicationException("Error: at least one broker must be specified in pageview_connection");

	if ( !doc["pageview_connection"]["topic"].isNull() ) {
        SetTopic(doc["pageview_connection"]["topic"].asString());
    }
	else
	{
        throw ApplicationException("Error: the topic must be specified in pageview_connection");
    }

    if ( !doc["pageview_connection"]["debug"].isNull() )
        kafkaDebug = doc["pageview_connection"]["debug"].asBool();

    // Database details
    if ( !doc["content_connection"]["hostname"].isNull() )
    	database.SetHostname(doc["content_connection"]["hostname"].asString());

    if ( !doc["content_connection"]["port"].isNull() ) {
    	try {
    		database.SetPort(doc["content_connection"]["port"].asUInt());
    	} catch ( const char *error ) {
			cerr << "WARN: Ignoring bad batch size value: " << error << endl;
		}
    }

    if ( !doc["content_connection"]["username"].isNull() )
        database.SetUsername(doc["content_connection"]["username"].asString());

    if ( !doc["content_connection"]["password"].isNull() )
        database.SetPassword(doc["content_connection"]["password"].asString());

    if ( !doc["content_connection"]["database"].isNull() )
        database.SetDatabase(doc["content_connection"]["database"].asString());
}

void ProcessCfg::SetBatchSize ( uint32_t b ) {
    batchCommitSize = b;
}
uint32_t ProcessCfg::GetBatchSize () {
    return batchCommitSize;
}

PostgresCfg *ProcessCfg::GetDatabaseCfg() {
	return &database;
}

vector<string> ProcessCfg::GetBrokers() {
    return brokers;
}

bool ProcessCfg::DebugEnabled() {
	return kafkaDebug;
}

void ProcessCfg::SetTopic ( string t ) {
    if ( t.empty() )
        throw ApplicationException("Topic may not be empty");

    topic = t;
}
string ProcessCfg::GetTopic () {
    return topic;
}

