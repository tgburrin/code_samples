/*
 * SourceReference.cpp
 *
 *  Created on: Oct 10, 2017
 *      Author: tgburrin
 */

#include "SourceReference.h"

SourceReference::SourceReference(string sid, PostgresCfg *c) {
	sourceId = sid;
	dbh = new PostgresDbh(c);
	isPrivateDbh = true;

	RefreshFromDatabase();
}

SourceReference::SourceReference(string sid, PostgresDbh *d) {
	sourceId = sid;
	dbh = d;
	isPrivateDbh = false;

	RefreshFromDatabase();
}

SourceReference::~SourceReference() {
	if ( isPrivateDbh )
		delete dbh;
}

void SourceReference::RefreshFromDatabase()
{
	sourceRef = 0;
	isNull = true;

	vector<string> args = {sourceId};
	uint64_t rows = dbh->ExecuteStatement("select * from get_source_ref($1)", args);
	if ( rows ) {
		vector< vector<char *> > tbl = dbh->GetRows();
		if ( tbl[0][0] != NULL ) {
			sourceRef = stoull(tbl[0][0]);
			isNull = false;
		}
	}
}

string SourceReference::GetId()
{
	return sourceId;
}

int64_t SourceReference::GetSourceRef()
{
	return sourceRef;
}

bool SourceReference::IsNull()
{
	return isNull;
}

void SourceReference::UpdateSourceRef(int64_t sr)
{
	vector<string> args = {sourceId, to_string(sr)};
	uint64_t rows = dbh->ExecuteStatement("select * from set_source_ref($1,$2)", args);
	if ( rows ) {
		vector< vector<char *> > tbl = dbh->GetRows();
		sourceRef = stoull(tbl[0][0]);
		isNull = false;
	}
	else
		throw ApplicationException("No rows returned while updating source reference "+sourceId);
}
