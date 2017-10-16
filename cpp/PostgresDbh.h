/*
 * PostgresDbh.h
 *
 *  Created on: Oct 10, 2017
 *      Author: tgburrin
 */

#ifndef POSTGRESDBH_H_
#define POSTGRESDBH_H_

#include <vector>

// https://www.postgresql.org/
#include <libpq-fe.h>

#include "PostgresCfg.h"
#include "ApplicationException.h"

using namespace std;

class PostgresDbh {
private:
	PGconn *dbh;
	PGresult *pgresult;
	bool inTxn;

public:
	PostgresDbh(PostgresCfg *);
	virtual ~PostgresDbh();

	void StartTxn();
	void CommitTxn();
	void RollbackTxn();

	bool IsTxnStarted();
	uint64_t ExecuteStatement(string, vector<string>);
	vector< vector<char *> > GetRows();
};

#endif /* POSTGRESDBH_H_ */
