/*
 * PostgresDbh.cpp
 *
 *  Created on: Oct 10, 2017
 *      Author: tgburrin
 */

#include "PostgresDbh.h"

PostgresDbh::PostgresDbh(PostgresCfg *c)
{
	inTxn = false;
    pgresult = 0;
    dbh = 0;

    if ( !PQisthreadsafe() )
    	throw ApplicationException("libpq is not thread safe");

	dbh = PQconnectdb(c->GetURI().c_str());

	if ( PQstatus(dbh) != CONNECTION_OK )
		throw ApplicationException(PQerrorMessage(dbh));
}

PostgresDbh::~PostgresDbh()
{
	if ( dbh )
		PQfinish(dbh);
}

void PostgresDbh::StartTxn()
{
	PGresult *pgresult = PQexec(dbh, "BEGIN");
    if ( PQresultStatus(pgresult) != PGRES_COMMAND_OK )
        throw ApplicationException("Error while starting transaction: " + string(PQerrorMessage(dbh)));

    PQclear(pgresult);
    inTxn = true;
}

void PostgresDbh::CommitTxn()
{
	if( !inTxn )
		throw ApplicationException("Cannot commit a transaction that is not started");

	PGresult *pgresult = PQexec(dbh, "COMMIT");
    if ( PQresultStatus(pgresult) != PGRES_COMMAND_OK )
        throw ApplicationException("Error while committing transaction: " + string(PQerrorMessage(dbh)));

    PQclear(pgresult);
    inTxn = false;
}

void PostgresDbh::RollbackTxn()
{
	if( !inTxn )
		throw ApplicationException("Cannot rollback a transaction that is not started");

	PGresult *pgresult = PQexec(dbh, "ROLLBACK");
    if ( PQresultStatus(pgresult) != PGRES_COMMAND_OK )
        throw ApplicationException("Error while rolling back transaction: " + string(PQerrorMessage(dbh)));

    PQclear(pgresult);
    inTxn = false;
}

bool PostgresDbh::IsTxnStarted()
{
	return inTxn;
}

uint64_t PostgresDbh::ExecuteStatement(string sql, vector<string> parameters)
{
	uint64_t rows_affected_returned = 0;
	bool single_txn = false;
	string resultStr;

	if ( !inTxn ) {
		single_txn = true;
		StartTxn();
	}

    if ( pgresult ) {
        PQclear(pgresult);
        pgresult = 0;
    }

	if ( !parameters.size() ) {
		pgresult = PQexec(dbh, sql.c_str());
	}
	else
	{
		const char *p[parameters.size()];
		for ( int i=0; i < parameters.size(); i++)
		{
			p[i] = parameters[i].c_str();
		}

		// Return in text format for now
		pgresult = PQexecParams(dbh,
								sql.c_str(),
								parameters.size(),
								NULL,
								p,
								NULL,
								NULL,
								0
							   );
	}


	if ( !pgresult )
		throw ApplicationException("Out of memory while producing result");

	switch (PQresultStatus(pgresult)) {
		case PGRES_TUPLES_OK:
			// This might overflow for result sets > 2bn according to documentation
			rows_affected_returned = PQntuples(pgresult);
			break;
		case PGRES_COMMAND_OK:
			resultStr = PQcmdTuples(pgresult);

			if ( !resultStr.empty() )
				rows_affected_returned = stoi(resultStr);

			break;
		case PGRES_FATAL_ERROR:
			throw ApplicationException("Error while executing statement: " + string(PQerrorMessage(dbh)) + "\nSQL: "+ sql);
			break;
		default:
			throw ApplicationException("Unhandled return value: " + string(PQerrorMessage(dbh)));
	}

	if (single_txn)
		CommitTxn();

	return rows_affected_returned;
}

vector< vector<char *> > PostgresDbh::GetRows()
{
	vector< vector<char *> > rv;
	if ( pgresult ) {
		for( int i=0; i < PQntuples(pgresult); i++ )
		{
			vector<char *> row(PQnfields(pgresult));
			for( int k=0; k < PQnfields(pgresult); k++ )
			{
				if ( PQgetisnull(pgresult, i, k) )
				{
                    row[k] = NULL;
				}
                else
                {
					row[k] = PQgetvalue(pgresult, i, k);
                }
			}
            rv.push_back(row);
		}
		PQclear(pgresult);
        pgresult = 0;
	}
	return rv;
}
