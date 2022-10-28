package net.tgburrin.pageview_svc.client;

import java.util.UUID;

import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Table;

import net.tgburrin.pageview_svc.InvalidDataException;

@Table(name = "client", schema = "pageview")
public class Client {
	@Id
	private UUID id;
	private String name;

	public UUID getId() {
		return id;
	}
	public String getName() {
		return name;
	}
	public void setName(String name) {
		this.name = name;
	}

	public void validateRecord() throws InvalidDataException {
		if(this.name == null || this.name.equals(""))
			throw new InvalidDataException("A valid group name must be specified");
	}
	public void mergeUpdate(Client nc) {
		if ( nc.name != null )
			name = nc.name;
	}
}
