/*
 * ContentPageViewConsumer.cpp
 *
 *  Created on: Oct 16, 2017
 *      Author: tgburrin
 */

#include "ContentPageViewConsumer.h"

ContentPageViewConsumer::ContentPageViewConsumer(ProcessCfg *c) : MessageConsumer(c) {
	sourceReferenceName = "content_pageview";
	sourceReference = new SourceReference(sourceReferenceName, dbh);
}

ContentPageViewConsumer::~ContentPageViewConsumer() {
	delete sourceReference;
}

void ContentPageViewConsumer::_ProcessMessage(Json::Value msg)
{
	if ( msg["type"].asString().compare("content_pageview") == 0 )
	{
		string id = msg["id"].asString();
		string pvdt = msg["pageview_dt"].asString();

		vector<string> args;
		args.push_back(id);
		args.push_back(pvdt);

		if ( debugOn )
			cout << "select * from content_pageview(\"" << id << "\", \"" << pvdt << "\")" << endl;

		uint64_t rows = dbh->ExecuteStatement("select * from content_pageview($1, $2)", args);
		if ( rows != 1 )
			cerr << rows << " rows were returned while doing a client content pageview" << endl;
	}
}
