/*
 * PostgresCfg.h
 *
 *  Created on: Oct 10, 2017
 *      Author: tgburrin
 */
#ifndef POSTGRESCFG_H_
#define POSTGRESCFG_H_

#include <string>
#include <sstream>

using namespace std;

class PostgresCfg {
private:
    string username;
    string password;
    string hostname;
    uint16_t port = 5432;
    string database;

public:
    PostgresCfg();
    virtual ~PostgresCfg();

    void SetDatabase(string);
    string GetDatabase(void);

    void SetUsername(string);
    string GetUsername(void);

    void SetPassword(string);
    string GetPassword(void);

    void SetHostname(string);
    string GetHostname(void);

    void SetPort(uint16_t);
    uint16_t GetPort(void);

    string GetURI(void);
};

#endif /* POSTGRESCFG_H_ */
