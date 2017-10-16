/*
 * ClientPageViewConsumer.h
 *
 *  Created on: Oct 16, 2017
 *      Author: tgburrin
 */

#ifndef CLIENTPAGEVIEWCONSUMER_H_
#define CLIENTPAGEVIEWCONSUMER_H_

#include "MessageConsumer.h"

using namespace std;

class ClientPageViewConsumer : public MessageConsumer {
private:
	void _ProcessMessage(Json::Value);

public:
	ClientPageViewConsumer(ProcessCfg *);
	virtual ~ClientPageViewConsumer();
};

#endif /* CLIENTPAGEVIEWCONSUMER_H_ */
