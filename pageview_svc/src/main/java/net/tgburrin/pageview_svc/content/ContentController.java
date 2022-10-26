package net.tgburrin.pageview_svc.content;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

@RestController
@RequestMapping(path = "/v{apiVersion}/content")
public class ContentController {
	@ResponseStatus(HttpStatus.CREATED)
	@PostMapping(value="/create", consumes = "application/json", produces = "application/json")
	public void addContent(@PathVariable Integer apiVersion) {
	}

	@ResponseStatus(HttpStatus.OK)
	@PutMapping(value="/update/{id}", consumes = "application/json", produces = "application/json")
	public void updateContent(@PathVariable Integer apiVersion, @PathVariable("name")  Integer contentId) {
	}
}
