package net.tgburrin.pageview_svc.pageview;

import java.util.UUID;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.databind.ObjectMapper;

import net.tgburrin.pageview_svc.InvalidDataException;
import net.tgburrin.pageview_svc.PageviewKafkaService;

@RestController
@RequestMapping(path = "/v{apiVersion}/pageview")
public class PageviewController {
	@Autowired
	private PageviewKafkaService pks;

    @Autowired
    private ObjectMapper objectMapper;

	@ResponseStatus(HttpStatus.NO_CONTENT)
	@GetMapping(value="/content/{id}", produces = "application/json")
	public void contentView(@PathVariable Integer apiVersion, @PathVariable("id")  UUID contentId) throws InvalidDataException {
		Pageview p = new Pageview(contentId);
		String payload = null;
		try {
			payload = objectMapper.writerWithDefaultPrettyPrinter().writeValueAsString(p);
		} catch (JsonProcessingException e) {
			throw new InvalidDataException(e.getMessage());
		}

		pks.sendMessage(payload);
	}
}
