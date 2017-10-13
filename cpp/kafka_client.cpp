#include <iostream>
#include <sstream>
#include <string>
#include <vector>
#include <thread>
#include <mutex>
#include <csignal>
#include <chrono>

#include <unistd.h>

#include "PostgresDbh.h"
#include "PostgresCfg.h"
#include "ProcessCfg.h"
#include "SourceReference.h"
#include "MessageConsumer.h"

using namespace std;

mutex counterLock;

uint64_t eventCounter = 0;
uint counterDisplaySeconds = 5;
uint commitTimeSeconds = 30;

bool running = true;

void IncrementCounter ()
{
	counterLock.lock();
	eventCounter++;
	counterLock.unlock();
}

void ProcessClientPageViews (ProcessCfg *cfg)
{
	PostgresDbh dbh(cfg->GetDatabaseCfg());
	SourceReference sr("client_pageview", &dbh);
    MessageConsumer *mc = 0;

    cout << "Creating client pv consumer" << endl;
    try {
    	mc = new MessageConsumer(*cfg);

    	if ( !sr.IsNull() )
    		mc->SetSourceReference(sr.GetSourceRef());

    	mc->Start();
    } catch ( ApplicationException &e ){
    	cerr << e.what() << endl;
    	exit(EXIT_FAILURE);
    }

    chrono::system_clock::time_point lastCommitTime = chrono::system_clock::now();
    uint64_t msgCounter = 0;

    while ( running )
    {
		chrono::system_clock::time_point n = chrono::system_clock::now();
    	string *message;

    	while ( (message = mc->Read()) != NULL )
		{
			cout << *message << endl;
			delete message;

			msgCounter++;
			IncrementCounter();

			cout << "Processing client message" << endl;

			if ( msgCounter % cfg->GetBatchSize() == 0 )
			{
				cout << "Committing based on batch size" << endl;
				sr.UpdateSourceRef(mc->GetSourceReference());
				lastCommitTime = n;
				msgCounter = 0;
			}
		}

		if ( msgCounter > 0 && chrono::duration_cast<chrono::seconds>(n - lastCommitTime).count() >= commitTimeSeconds )
		{
			cout << "Committing based on time" << endl;
			sr.UpdateSourceRef(mc->GetSourceReference());
			lastCommitTime = n;
			msgCounter = 0;
		}
    	usleep(0);
    }

    delete mc;
}
void ProcessContentPageViews (ProcessCfg *cfg)
{
	PostgresDbh dbh(cfg->GetDatabaseCfg());
	cout << "Getting sourceRef" << endl;
	SourceReference sr("content_pageview", &dbh);
    MessageConsumer *mc = 0;

    chrono::system_clock::time_point ct = chrono::system_clock::now();

    cout << "Creating content pv consumer" << endl;
    try {
    	mc = new MessageConsumer(*cfg);
    	if ( !sr.IsNull() )
    		mc->SetSourceReference(sr.GetSourceRef());

    	mc->Start();
    } catch ( ApplicationException &e ){
    	cerr << e.what() << endl;
    	exit(EXIT_FAILURE);
    }

    chrono::system_clock::time_point lastCommitTime = chrono::system_clock::now();
    uint64_t msgCounter = 0;

    while ( running )
    {
		chrono::system_clock::time_point n = chrono::system_clock::now();
    	string *message;

    	while ( (message = mc->Read()) != NULL )
		{
			cout << *message << endl;
			delete message;

			msgCounter++;
			IncrementCounter();

			cout << "Processing client message" << endl;

			if ( msgCounter % cfg->GetBatchSize() == 0 )
			{
				cout << "Committing based on batch size" << endl;
				sr.UpdateSourceRef(mc->GetSourceReference());
				lastCommitTime = n;
				msgCounter = 0;
			}
		}

		if ( msgCounter > 0 && chrono::duration_cast<chrono::seconds>(n - lastCommitTime).count() >= commitTimeSeconds )
		{
			cout << "Committing based on time" << endl;
			sr.UpdateSourceRef(mc->GetSourceReference());
			lastCommitTime = n;
			msgCounter = 0;
		}
    }

    delete mc;
}

void DisplayEventsProcessed () {
	counterLock.lock();
	double rate = eventCounter / (double)counterDisplaySeconds;
	cout << eventCounter << " events processed (" << rate << "/s)" << endl;
	eventCounter = 0;
	counterLock.unlock();
}

void GracefulShutdown (int signal) {
	cout << "Exiting..." << endl;
    running = false;
}

string PrintUsage( string program_name ) {
    stringstream message;

    message << "Usage: " << program_name << " <options>" << endl;
    message << "\t-c <config file> (required)" << endl;
    message << "\t-h this help menu" << endl;

    return message.str();
}

int main (int argc, char **argv) {
    string configFile;

    int opt;
    extern char *optarg;
    extern int optind, opterr, optopt;

    while ((opt = getopt(argc, argv, "hc:k")) != -1) {
        switch (opt) {
            case 'c':
                configFile = optarg;
                break;
            case 'h':
                cout << PrintUsage(basename(argv[0]));
                exit(EXIT_SUCCESS);
            default:
                cerr << "Unknown option provided" << endl;
                cerr << PrintUsage(basename(argv[0]));
                exit(EXIT_FAILURE);
        }
    }

    if ( configFile.empty() ) {
        cerr << "-c is a required option" << endl;
        cerr << PrintUsage(basename(argv[0]));
        exit(EXIT_FAILURE);
    }

    ProcessCfg cfg(configFile);

    signal(SIGINT, GracefulShutdown);

    vector<thread> children(2);
    children.at(0) = thread(ProcessClientPageViews, &cfg);
    children.at(1) = thread(ProcessContentPageViews, &cfg);

    while ( running )
    {
    	DisplayEventsProcessed();
    	sleep(counterDisplaySeconds);
    }

    for ( uint i = 0; i < children.size(); i++ )
    	if (children.at(i).joinable() )
    		children.at(i).join();

    exit(EXIT_SUCCESS);
}
