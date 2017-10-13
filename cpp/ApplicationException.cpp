/*
 * ApplicationException.cpp
 *
 *  Created on: Oct 13, 2017
 *      Author: tgburrin
 */

#include "ApplicationException.h"

ApplicationException::ApplicationException(string m): message(m) {}
ApplicationException::ApplicationException(const char *m): message(m) {}
ApplicationException::~ApplicationException() throw () {}

const char* ApplicationException::what()
{
	return message.c_str();
}
