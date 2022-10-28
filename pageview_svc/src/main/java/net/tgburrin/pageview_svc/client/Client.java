package net.tgburrin.pageview_svc.client;

import java.util.UUID;

import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Table;

import net.tgburrin.pageview_svc.DatabaseRecordInterface;
import net.tgburrin.pageview_svc.InvalidDataException;

@Table(name = "client")
public class Client implements DatabaseRecordInterface<Client> {
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

	@Override
	public void validateRecord() throws InvalidDataException {
		if(this.name == null || this.name.equals(""))
			throw new InvalidDataException("A valid group name must be specified");
	}

	@Override
	public void mergeUpdate(Client newRecord) {
		if ( newRecord.name != null )
			name = newRecord.name;
	}
}
