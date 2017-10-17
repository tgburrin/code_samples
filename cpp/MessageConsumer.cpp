/*
 * MessageConsumer.cpp
 *
 *  Created on: Oct 13, 2017
 *      Author: tgburrin
 */

#include "MessageConsumer.h"

MessageConsumer::MessageConsumer(ProcessCfg *c) {
	debugOn = false;
	commitTimeSeconds = 30;

	eventCounter = 0;
	eventCounterLock = 0;

	cfg = c;
	dbh = new PostgresDbh(cfg->GetDatabaseCfg());
}
MessageConsumer::~MessageConsumer() {
	delete dbh;
}

void MessageConsumer::_IncrementCounter(){
	if ( eventCounterLock != NULL && eventCounter != NULL ) {
		eventCounterLock->lock();
		*eventCounter += 1;
		eventCounterLock->unlock();
	}
}

void MessageConsumer::RunProcess(bool *running) {
    KafkaClient *kc = 0;

	kc = new KafkaClient(cfg);

	try {
		if ( !sourceReference->IsNull() )
			kc->SetSourceReference(sourceReference->GetSourceRef());

		kc->Start();
	} catch (ApplicationException &e) {
		stringstream newmsg;
		newmsg << e.what() << " for " << sourceReference->GetId();
		throw ApplicationException(newmsg.str());
	}

    chrono::system_clock::time_point lastCommitTime = chrono::system_clock::now();
    uint64_t msgCounter = 0;

    dbh->StartTxn();
    while ( *running )
    {
		chrono::system_clock::time_point n = chrono::system_clock::now();
    	string *message;

    	while ( (message = kc->Read()) != NULL )
		{
			//cout << *message << endl;

			try {
				Json::Value doc;
				Json::Reader rd;

				rd.parse(*message, doc, false);

				_ProcessMessage(doc);
			} catch ( Json::LogicError &e ) {
				cerr << "Could not process message" << endl;
				cerr << e.what() << endl;
				cerr << *message << endl;
			}
			_IncrementCounter();
			delete message;

			msgCounter++;
			if ( msgCounter % cfg->GetBatchSize() == 0 )
			{
				cout << "Committing based on batch size" << endl;
				sourceReference->UpdateSourceRef(kc->GetSourceReference());
				dbh->CommitTxn();
				lastCommitTime = n;
				msgCounter = 0;
				dbh->StartTxn();
			}
		}

		if ( msgCounter > 0 && chrono::duration_cast<chrono::seconds>(n - lastCommitTime).count() >= commitTimeSeconds )
		{
			cout << "Committing based on time" << endl;
			sourceReference->UpdateSourceRef(kc->GetSourceReference());
			dbh->CommitTxn();
			lastCommitTime = n;
			msgCounter = 0;
			dbh->StartTxn();
		} else if ( msgCounter == 0 ) {
			lastCommitTime = n;
		}

    	usleep(0);
    }

    if ( dbh->IsTxnStarted() )
    	dbh->RollbackTxn();

    kc->Stop();
    delete kc;
}

void MessageConsumer::AddCounters(uint64_t *evc, mutex *evcl)
{
	eventCounter = evc;
	eventCounterLock = evcl;
}
