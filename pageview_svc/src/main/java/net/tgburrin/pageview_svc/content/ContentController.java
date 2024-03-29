package net.tgburrin.pageview_svc.content;

import java.util.ArrayList;
import java.util.UUID;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.PutMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import net.tgburrin.pageview_svc.ContentPageviewService;
import net.tgburrin.pageview_svc.InvalidRecordException;

@RestController
@RequestMapping(path = "/v{apiVersion}/content")
public class ContentController {
	@Autowired
	ContentPageviewService pvsvc;

	@ResponseStatus(HttpStatus.CREATED)
	@PostMapping(value="/maintain", consumes = "application/json", produces = "application/json")
	public void addContent(@PathVariable Integer apiVersion, @RequestBody Content c) {

	}

	@ResponseStatus(HttpStatus.OK)
	@PutMapping(value="/maintain/{id}", consumes = "application/json", produces = "application/json")
	public void updateContent(@PathVariable Integer apiVersion, @PathVariable("id")  UUID contentId, @RequestBody Content newContent) {
	}

	@ResponseStatus(HttpStatus.OK)
	@GetMapping(value="/read/find/{id}", produces = "application/json")
	public Content getContent(@PathVariable Integer apiVersion, @PathVariable("id")  UUID contentId) throws InvalidRecordException  {
		return pvsvc.findContentById(contentId);
	}

	@ResponseStatus(HttpStatus.OK)
	@GetMapping(value="/read/list", produces = "application/json")
	public ArrayList<Content> listContent(@PathVariable Integer apiVersion) {
		return pvsvc.listAllContent();
	}
}
