/*
 * PostgresCfg.cpp
 *
 *  Created on: Oct 10, 2017
 *      Author: tgburrin
 */

#include "PostgresCfg.h"

PostgresCfg::PostgresCfg() {
}

PostgresCfg::~PostgresCfg() {
}

void PostgresCfg::SetDatabase (string d) {
    database = d;
}
string PostgresCfg::GetDatabase () {
    return database;
}

void PostgresCfg::SetUsername (string u) {
    username = u;
}
string PostgresCfg::GetUsername () {
    return username;
}

void PostgresCfg::SetPassword (string p) {
    password = p;
}
string PostgresCfg::GetPassword () {
    return password;
}

void PostgresCfg::SetHostname (string h) {
    hostname = h;
}
string PostgresCfg::GetHostname () {
    return hostname;
}

void PostgresCfg::SetPort (uint16_t p) {
    port = p;
}
uint16_t PostgresCfg::GetPort () {
    return port;
}

string PostgresCfg::GetURI (void) {
    stringstream uri;

    uri << "postgres://";
    if ( !username.empty() ) {
        uri << username;
        if ( !password.empty() )
            uri << ":" << password;

        uri << "@";
    }

    uri << (!hostname.empty() ? hostname : "localhost");
    if ( port > 0 )
        uri << ":" << port;

    if ( !database.empty() )
        uri << "/" << database;

    return uri.str();
}
