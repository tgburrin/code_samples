#include <iostream>
#include <sstream>
#include <string>
#include <vector>
#include <thread>
#include <mutex>
#include <csignal>

#include <unistd.h>

#include "ProcessCfg.h"
#include "ApplicationException.h"
#include "ClientPageViewConsumer.h"
#include "ContentPageViewConsumer.h"

using namespace std;

mutex counterLock;

uint64_t eventCounter = 0;
uint counterDisplaySeconds = 5;

bool running = true;

void IncrementCounter ()
{
	counterLock.lock();
	eventCounter++;
	counterLock.unlock();
}

// We could also do a static copy of RunProcess() and allow thread to call that directly
void ProcessClientPageViews (ProcessCfg *cfg)
{
	try {
		ClientPageViewConsumer cpv(cfg);
		cpv.RunProcess(&running);
	} catch (ApplicationException &e) {
		cerr << e.what() << endl;
		exit(EXIT_FAILURE);
	}

}
void ProcessContentPageViews (ProcessCfg *cfg)
{
	try {
		ContentPageViewConsumer cpv(cfg);
		cpv.RunProcess(&running);
	} catch (ApplicationException &e) {
		cerr << e.what() << endl;
		exit(EXIT_FAILURE);
	}
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

    ProcessCfg *cfg;
    try {
    	cfg = new ProcessCfg(configFile);
    } catch (ApplicationException &e) {
    	cerr << e.what() << endl;
    	exit(EXIT_FAILURE);
    }

    signal(SIGINT, GracefulShutdown);

    vector<thread> children(2);
    children.at(0) = thread(ProcessContentPageViews, cfg);
    children.at(1) = thread(ProcessClientPageViews, cfg);

    while ( running )
    {
    	DisplayEventsProcessed();
    	sleep(counterDisplaySeconds);
    }

    for ( uint i = 0; i < children.size(); i++ )
    	if (children.at(i).joinable() )
    		children.at(i).join();

    delete cfg;
    exit(EXIT_SUCCESS);
}
