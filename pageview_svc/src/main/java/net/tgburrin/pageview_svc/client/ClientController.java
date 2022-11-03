package net.tgburrin.pageview_svc.client;

import java.util.UUID;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.PatchMapping;
import org.springframework.web.bind.annotation.PathVariable;
import org.springframework.web.bind.annotation.PostMapping;
import org.springframework.web.bind.annotation.RequestBody;
import org.springframework.web.bind.annotation.RequestMapping;
import org.springframework.web.bind.annotation.ResponseStatus;
import org.springframework.web.bind.annotation.RestController;

import net.tgburrin.pageview_svc.ContentPageviewService;
import net.tgburrin.pageview_svc.InvalidDataException;
import net.tgburrin.pageview_svc.InvalidRecordException;

@RestController
@RequestMapping(path = "/v{apiVersion}/client")
public class ClientController {
	@Autowired
	private ContentPageviewService pvsvc;

	@ResponseStatus(HttpStatus.CREATED)
	@PostMapping(value="/maintain", consumes = "application/json", produces = "application/json")
	public Client addClient(@PathVariable Integer apiVersion, @RequestBody Client newClient) throws InvalidDataException, InvalidRecordException {
		Client c = pvsvc.findClientByName(newClient.getName());
		if ( c != null )
			throw new InvalidDataException("Existing client with name "+newClient.getName());
		return pvsvc.addClient(newClient);
	}

	@ResponseStatus(HttpStatus.OK)
	@PatchMapping(value="/maintain/{id}", consumes = "application/json", produces = "application/json")
	public Client updateClient(@PathVariable Integer apiVersion, @PathVariable("id")  UUID clientId, @RequestBody Client newClient) throws InvalidRecordException, InvalidDataException {
		Client ec = pvsvc.findClientById(clientId);
		ec.mergeUpdate(newClient);
		return pvsvc.updateClient(ec);
	}

	@ResponseStatus(HttpStatus.OK)
	@GetMapping(value="/read/find/{id}", produces = "application/json")
	public Client updateClient(@PathVariable Integer apiVersion, @PathVariable("id")  UUID clientId) throws InvalidRecordException {
		return pvsvc.findClientById(clientId);
	}
}
