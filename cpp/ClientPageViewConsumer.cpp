/*
 * ClientPageViewConsumer.cpp
 *
 *  Created on: Oct 16, 2017
 *      Author: tgburrin
 */

#include "ClientPageViewConsumer.h"

ClientPageViewConsumer::ClientPageViewConsumer(ProcessCfg *c) : MessageConsumer(c) {
    sourceReferenceName = "client_pageview";
    sourceReference = new SourceReference(sourceReferenceName, dbh);
}

ClientPageViewConsumer::~ClientPageViewConsumer() {
    delete sourceReference;
}

void ClientPageViewConsumer::_ProcessMessage(Json::Value msg)
{
    if ( msg["type"].asString().compare("content_pageview") == 0 )
    {
        string content_id = msg["content_id"].asString();
        string pvdt = msg["pageview_dt"].asString();

        vector<string> args;
        args.push_back(content_id);
        args.push_back(pvdt);

        if ( debugOn )
            cout << "select * from pageview.client_content_pageview(\"" << content_id << "\", \"" << pvdt << "\")" << endl;

        uint64_t rows = dbh->ExecuteStatement("select * from pageview.client_content_pageview($1, $2)", args);
        if ( rows != 1 )
            cerr << rows << " rows were returned while doing a client content pageview" << endl;
    }
}
