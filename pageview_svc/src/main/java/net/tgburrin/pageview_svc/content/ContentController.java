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

@RestController
@RequestMapping(path = "/v{apiVersion}/content")
public class ContentController {
	@Autowired
	ContentRepository contentRepo;

	@ResponseStatus(HttpStatus.CREATED)
	@PostMapping(value="/create", consumes = "application/json", produces = "application/json")
	public void addContent(@PathVariable Integer apiVersion, @RequestBody Content c) {
	}

	@ResponseStatus(HttpStatus.OK)
	@PutMapping(value="/update/{id}", consumes = "application/json", produces = "application/json")
	public void updateContent(@PathVariable Integer apiVersion, @PathVariable("id")  UUID contentId, @RequestBody Content newContent) {
	}

	@ResponseStatus(HttpStatus.OK)
	@GetMapping(value="/list", produces = "application/json")
	public ArrayList<Content> listContent(@PathVariable Integer apiVersion) {
		ArrayList<Content> rv = new ArrayList<Content>();
		for ( Content c : contentRepo.findAll() )
			rv.add(c);
		return rv;
	}
}