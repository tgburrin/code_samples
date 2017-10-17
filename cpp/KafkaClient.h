/*
 * KafkaClient.h
 *
 *  Created on: Oct 16, 2017
 *      Author: tgburrin
 */

#ifndef KAFKACLIENT_H_
#define KAFKACLIENT_H_

#include <iostream>
#include <cstring>

#include <librdkafka/rdkafkacpp.h>

#include "ApplicationException.h"
#include "ProcessCfg.h"

using namespace std;

class KafkaClient {
private:
	int64_t sourceReference;
	string topic_name;
	uint32_t partition;

	string message;

	bool running;
	bool debug;

	RdKafka::Conf *conf;
	RdKafka::Conf *tconf;

	RdKafka::Consumer *consumer;
	RdKafka::Topic *topic;

	bool _ProcessMessage(RdKafka::Message *);

public:
	KafkaClient(ProcessCfg *);
	virtual ~KafkaClient();

	void SetDebug(bool);
	void SetSourceReference(int64_t);
	int64_t GetSourceReference();
	void Start();
	void Stop();
	string *Read();
};

#endif /* KAFKACLIENT_H_ */
