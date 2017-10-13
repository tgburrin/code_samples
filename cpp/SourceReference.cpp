/*
 * SourceReference.cpp
 *
 *  Created on: Oct 10, 2017
 *      Author: tgburrin
 */

#include "SourceReference.h"

SourceReference::SourceReference(string sid, PostgresCfg c) {
	sourceId = sid;
	PostgresDbh tmp(c);
	dbh = &tmp;
	isPrivateDbh = true;

	_RefreshFromDatabase();
}

SourceReference::SourceReference(string sid, PostgresDbh *d) {
	sourceId = sid;
	dbh = d;
	isPrivateDbh = false;

	_RefreshFromDatabase();
}

SourceReference::~SourceReference() {
}

void SourceReference::_RefreshFromDatabase()
{
	vector<string> args = {sourceId};
	uint64_t rows = dbh->ExecuteStatement("select * from get_source_ref($1)", args);
	if ( rows ) {
		vector< vector<char *> > tbl = dbh->GetRows();
		sourceRef = boost::lexical_cast<int64_t>(tbl[0][0]);
		isNull = false;
	}
	else
	{
		sourceRef = 0;
		isNull = true;
	}
}

int64_t SourceReference::GetSourceRef()
{
	_RefreshFromDatabase();
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
		sourceRef = boost::lexical_cast<uint64_t>(tbl[0][0]);
		isNull = false;
	}
	else
		throw ApplicationException("No rows returned while updating source reference "+sourceId);
}
