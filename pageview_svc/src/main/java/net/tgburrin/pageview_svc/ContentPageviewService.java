package net.tgburrin.pageview_svc;

import java.util.ArrayList;
import java.util.Optional;
import java.util.UUID;

import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.stereotype.Service;

import net.tgburrin.pageview_svc.client.Client;
import net.tgburrin.pageview_svc.client.ClientRepository;
import net.tgburrin.pageview_svc.content.Content;
import net.tgburrin.pageview_svc.content.ContentRepository;

@Service
public class ContentPageviewService {
	@Autowired
	private ClientRepository clientRepository;
	@Autowired
	private ContentRepository contentRepository;

	public Client addClient(Client c) throws InvalidDataException, InvalidRecordException {
		c.validateRecord();
		return saveClient(c);
	}

	public Client findClientByName(String name) {
		return clientRepository.findByName(name);
	}

	public Client findClientById(UUID id) throws InvalidRecordException {
		Optional<Client> c = clientRepository.findById(id);
		if ( !c.isPresent() )
			throw new InvalidRecordException("Client "+id+" could not be located");
		return c.get();
	}

	public Client updateClientName(UUID id, Client newClient) throws InvalidDataException, InvalidRecordException {
		Client c = findClientById(id);
		if ( c == null )
			throw new InvalidDataException("Client id "+id+" coult not be found");

		if ( newClient.getName() == null || newClient.getName().equals("") )
			throw new InvalidDataException("Client name may not be blank when updated");

		c.setName(newClient.getName());
		return saveClient(c);
	}

	public Client updateClient(Client c) throws InvalidDataException, InvalidRecordException {
		return saveClient(c);
	}

	public Client saveClient(Client c) throws InvalidDataException, InvalidRecordException {
		if ( c == null )
			throw new InvalidDataException("Invalid Client provided");

		c.validateRecord();
		return clientRepository.save(c); // this is done to allow triggers to add data
	}

	public Content addContent(Content newContent) {
		return null;
	}

	public Content findContentById(UUID id) throws InvalidRecordException {
		Optional<Content> c = contentRepository.findById(id);
		if ( !c.isPresent() )
			throw new InvalidRecordException("Content "+id+" could not be located");
		return c.get();
	}

	public ArrayList<Content> listAllContent() {
		ArrayList<Content> rv = new ArrayList<Content>();
		for (Content c : contentRepository.findAll() )
			rv.add(c);
		return rv;
	}
}
