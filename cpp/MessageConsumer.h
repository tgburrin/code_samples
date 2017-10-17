/*
 * MessageConsumer.h
 *
 *  Created on: Oct 13, 2017
 *      Author: tgburrin
 */

#ifndef MESSAGECONSUMER_H_
#define MESSAGECONSUMER_H_

#include <mutex>

#include <unistd.h>

#include <json/json.h>
#include <json/reader.h>

#include "ApplicationException.h"
#include "KafkaClient.h"
#include "PostgresDbh.h"
#include "SourceReference.h"

using namespace std;

class MessageConsumer {
private:
	void _IncrementCounter();

protected:
	uint64_t *eventCounter;
	mutex *eventCounterLock;

	bool debugOn;

	string sourceReferenceName;

	ProcessCfg *cfg;
	PostgresDbh *dbh;
	SourceReference *sourceReference;

	uint commitTimeSeconds;

	virtual void _ProcessMessage(Json::Value) = 0;

public:
	MessageConsumer(ProcessCfg *c);
	virtual ~MessageConsumer();

	void RunProcess(bool *);
	void AddCounters(uint64_t *, mutex *);
};

#endif /* MESSAGECONSUMER_H_ */
