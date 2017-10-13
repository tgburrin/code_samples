/*
 * MessageConsumer.h
 *
 *  Created on: Oct 13, 2017
 *      Author: tgburrin
 */

#ifndef MESSAGECONSUMER_H_
#define MESSAGECONSUMER_H_

#include <string>
#include <sstream>
#include <vector>
#include <exception>

#include <librdkafka/rdkafkacpp.h>

#include "ApplicationException.h"
#include "ProcessCfg.h"

using namespace std;

class MessageConsumer {
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
	MessageConsumer(ProcessCfg);
	virtual ~MessageConsumer();

	void SetDebug(bool);
	void SetSourceReference(int64_t);
	int64_t GetSourceReference();
	void Start();
	void Stop();
	string *Read();
};

#endif /* MESSAGECONSUMER_H_ */
