/*
 * ApplicationException.h
 *
 *  Created on: Oct 13, 2017
 *      Author: tgburrin
 */

#ifndef APPLICATIONEXCEPTION_H_
#define APPLICATIONEXCEPTION_H_

#include <string>
#include <exception>

using namespace std;

class ApplicationException : exception {
private:
	string message;

public:
	ApplicationException(string);
	ApplicationException(const char *);
	virtual ~ApplicationException() throw();
	virtual const char* what();
};

#endif /* APPLICATIONEXCEPTION_H_ */
