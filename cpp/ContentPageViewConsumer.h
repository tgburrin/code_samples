/*
 * ContentPageViewConsumer.h
 *
 *  Created on: Oct 16, 2017
 *      Author: tgburrin
 */

#ifndef CONTENTPAGEVIEWCONSUMER_H_
#define CONTENTPAGEVIEWCONSUMER_H_

#include "MessageConsumer.h"

using namespace std;

class ContentPageViewConsumer : public MessageConsumer {
private:
	void _ProcessMessage(Json::Value);

public:
	ContentPageViewConsumer(ProcessCfg *);
	virtual ~ContentPageViewConsumer();
};

#endif /* CONTENTPAGEVIEWCONSUMER_H_ */
