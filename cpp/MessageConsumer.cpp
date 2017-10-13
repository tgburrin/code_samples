/*
 * MessageConsumer.cpp
 *
 *  Created on: Oct 13, 2017
 *      Author: tgburrin
 */

#include "MessageConsumer.h"

MessageConsumer::MessageConsumer(ProcessCfg cfg) {
	debug = false;
	running = false;
	sourceReference = RdKafka::Topic::OFFSET_BEGINNING;
	partition = 0;
	conf = 0;
	tconf = 0;
	consumer = 0;
	topic = 0;

	topic_name = cfg.GetTopic();

	string err;

	// Broad configuration items
	conf = RdKafka::Conf::create(RdKafka::Conf::CONF_GLOBAL);

	stringstream bl;
	vector<string> brokers = cfg.GetBrokers();
	for(uint i = 0; i < brokers.size(); i++ )
	{
	  if ( i == 0 )
		  bl << ",";
	  bl << brokers[i];
	}

	conf->set("metadata.broker.list", bl.str(), err);
	if ( !err.empty() )
		throw ApplicationException(err);

	conf->set("group.id", "message_consumer", err);
	if ( !err.empty() )
		throw ApplicationException(err);

	conf->set("enable.auto.offset.store", "false", err);
	if ( !err.empty() )
		throw ApplicationException(err);

	conf->set("offset.store.method", "none", err);
	if ( !err.empty() ) {
		cerr << err << endl;
		throw ApplicationException(err);
	}

	if ( debug )
	{
		conf->set("debug", "all", err);
		if ( !err.empty() )
			throw ApplicationException(err);
	}

	// Topic configuration items
	tconf = RdKafka::Conf::create(RdKafka::Conf::CONF_TOPIC);

	tconf->set("auto.offset.reset", "beginning", err);
	if ( !err.empty() )
		throw ApplicationException(err);

	tconf->set("auto.commit.enable", "false", err);
	if ( !err.empty() )
		throw ApplicationException(err);

}

MessageConsumer::~MessageConsumer() {}

void MessageConsumer::SetDebug(bool b) {
	debug = b;
}

int64_t MessageConsumer::GetSourceReference()
{
	return sourceReference;
}

void MessageConsumer::SetSourceReference(int64_t sr)
{
	sourceReference = sr;
}

bool MessageConsumer::_ProcessMessage(RdKafka::Message *msg)
{
	bool messageProcessed = false;

	switch (msg->err()) {
		case RdKafka::ERR__TIMED_OUT:
			break;

		case RdKafka::ERR_NO_ERROR:
			if ( msg->len() )
			{
				message = string((char *)msg->payload());
			}
			else
			{
				message.clear();
			}

			sourceReference = msg->offset();
			messageProcessed = true;
			break;

		case RdKafka::ERR__PARTITION_EOF:
			break;

		// It might be worth it to handle the following separately in some better manner
		case RdKafka::ERR__UNKNOWN_TOPIC:
		case RdKafka::ERR__UNKNOWN_PARTITION:
		default:
			delete msg;
			throw ApplicationException(msg->errstr());
	}

	delete msg;
	return messageProcessed;
}

void MessageConsumer::Start() {
	string err;

	consumer = RdKafka::Consumer::create(conf, err);
	if ( !err.empty() )
		throw ApplicationException(err);

	topic = RdKafka::Topic::create(consumer, topic_name, tconf, err);
	if ( !err.empty() )
		throw ApplicationException(err);

	if ( sourceReference != RdKafka::Topic::OFFSET_BEGINNING )
	{
		if ( debug )
			cout << "Finding starting point from source reference " << sourceReference << endl;

		int64_t current_sr = sourceReference;

		RdKafka::ErrorCode errmsg = consumer->start(topic, partition, sourceReference);
		if (errmsg != RdKafka::ERR_NO_ERROR)
			throw ApplicationException(RdKafka::err2str(errmsg));

		RdKafka::Message *msg = consumer->consume(topic, partition, 1000);  // Wait 1s for the initial message;
		if ( _ProcessMessage(msg) && debug )
			cout << message << endl;

		if ( current_sr != sourceReference )
			throw ApplicationException("Starting message " + to_string(current_sr) + " could not be found");

	} else {
		RdKafka::ErrorCode errmsg = consumer->start(topic, partition, RdKafka::Topic::OFFSET_BEGINNING);
		if (errmsg != RdKafka::ERR_NO_ERROR)
			throw ApplicationException(RdKafka::err2str(errmsg));
	}
}

string *MessageConsumer::Read()
{
	string *rv = 0;

	RdKafka::Message *msg = consumer->consume(topic, partition, 0);
	if ( _ProcessMessage(msg) )
	{
		if ( debug )
			cout << message << endl;

		rv = new string(message);
	}

	return rv;
}

