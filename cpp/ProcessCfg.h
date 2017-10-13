/*
 * ProcessCfg.h
 *
 *  Created on: Oct 10, 2017
 *      Author: tgburrin
 */

#ifndef PROCESSCFG_H_
#define PROCESSCFG_H_

#include <string>
#include <cstdint>
#include <vector>

#include <fstream>
#include <sstream>

#include <boost/property_tree/ptree.hpp>
#include <boost/property_tree/json_parser.hpp>

#include "ApplicationException.h"
#include "PostgresCfg.h"

using namespace std;
namespace pt = boost::property_tree;

class ProcessCfg {
private:
	int batchCommitSize;
	vector<string> brokers;
	string topic;
	bool kafkaDebug;
    PostgresCfg database;

	void _ParseConfig(string);

public:
	ProcessCfg(string);
	virtual ~ProcessCfg();

	void SetBatchSize(uint32_t);
	uint32_t GetBatchSize(void);

	void SetTopic(string);
	string GetTopic(void);

	PostgresCfg GetDatabaseCfg(void);
	vector<string> GetBrokers(void);

	bool DebugEnabled();
};

#endif /* PROCESSCFG_H_ */
