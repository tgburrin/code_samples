package net.tgburrin.pageview_svc;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.kafka.core.KafkaTemplate;
import org.springframework.stereotype.Service;

@Service
public class PageviewKafkaService {
    @Autowired
    private KafkaTemplate<String, String> kafkaTemplate;
    private final static String topic = "pageview";

    public void sendMessage(String message)
    {
        this.kafkaTemplate.send(topic, message);
    }
}
