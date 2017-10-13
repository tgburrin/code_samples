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
    stringstream cf;
    string line;

    ifstream infile(config_file.c_str());

    while(getline(infile, line))
        cf << line << endl;

    infile.close();

    pt::ptree mytree;
    pt::read_json(cf, mytree);

    try {
        SetBatchSize(mytree.get<int>("batch_commit_size"));
    }
    catch ( pt::ptree_bad_path &err ) {}
    catch ( const char *error ) {
        cerr << "WARN: Ignoring bad batch size value: " << error << endl;
    }

    // Kafka details
    try {
        pt::ptree brk = mytree.get_child("pageview_connection.brokers");
        for(pt::ptree::const_iterator it = brk.begin(); it != brk.end(); it++) {
            string val = it->second.get_value<string>();
            if ( !val.empty() )
                brokers.push_back(it->second.get_value<string>());
        }

        if ( brokers.size() == 0 )
            throw ApplicationException("Error: at least one broker must be specified in pageview_connection");
    }
    catch ( pt::ptree_bad_path &err ) {
        throw ApplicationException("Error: brokers section of pageview_connection is required");
    }

    try {
        SetTopic(mytree.get<string>("pageview_connection.topic"));
    } catch ( pt::ptree_bad_path &err ) {
        throw ApplicationException("Error: the topic must be specified in pageview_connection");
    }

    try {
        kafkaDebug = mytree.get<bool>("pageview_connection.debug");
    } catch ( pt::ptree_bad_path &err ) {}

    // Database details
    try {
        database.SetHostname(mytree.get<string>("content_connection.hostname"));
    } catch ( pt::ptree_bad_path &err ) {}

    try {
        database.SetPort(mytree.get<int>("content_connection.port"));
    }
    catch ( pt::ptree_bad_path &err ) {}
    catch ( const char *error ) {
        cerr << "WARN: Ignoring bad batch size value: " << error << endl;
    }

    try {
        database.SetUsername(mytree.get<string>("content_connection.username"));
    } catch ( pt::ptree_bad_path &err ) {}

    try {
        database.SetPassword(mytree.get<string>("content_connection.password"));
    } catch ( pt::ptree_bad_path &err ) {}

    try {
        database.SetDatabase(mytree.get<string>("content_connection.database"));
    } catch ( pt::ptree_bad_path &err ) {}

    /*
    pt::ptree::const_iterator end = mytree.end();
    for(pt::ptree::const_iterator it = mytree.begin(); it != end; ++it)
        cout << it->first << ": " << it->second.get_value<string>() << endl;
    */
}

void ProcessCfg::SetBatchSize ( uint32_t b ) {
    batchCommitSize = b;
}
uint32_t ProcessCfg::GetBatchSize () {
    return batchCommitSize;
}

PostgresCfg ProcessCfg::GetDatabaseCfg() {
	return database;
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

