/*
 * SourceReference.h
 *
 *  Created on: Oct 10, 2017
 *      Author: tgburrin
 */

#ifndef SOURCEREFERENCE_H_
#define SOURCEREFERENCE_H_

#include <string>
#include <exception>

#include <boost/lexical_cast.hpp>

#include "ApplicationException.h"
#include "PostgresCfg.h"
#include "PostgresDbh.h"

using namespace std;

class SourceReference {
private:
	string sourceId;
	PostgresDbh *dbh;
	bool isPrivateDbh;

	int64_t sourceRef;
	bool isNull;


public:
	SourceReference(string, PostgresCfg);
	SourceReference(string, PostgresDbh *);
	virtual ~SourceReference();

	void RefreshFromDatabase();
	void UpdateSourceRef(int64_t);

	int64_t GetSourceRef();
	bool IsNull();
};

#endif /* SOURCEREFERENCE_H_ */
