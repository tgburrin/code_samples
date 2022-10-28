package net.tgburrin.pageview_svc.content;

import java.net.MalformedURLException;
import java.net.URL;
import java.util.UUID;

import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Table;

import net.tgburrin.pageview_svc.DatabaseRecordInterface;
import net.tgburrin.pageview_svc.InvalidDataException;

@Table(name = "content")
public class Content implements DatabaseRecordInterface<Content> {
	@Id
	private UUID id;

	private String name;
	private String url;

	public UUID getId() {
		return id;
	}

	public String getName() {
		return name;
	}
	public void setName(String name) {
		this.name = name;
	}
	public URL getUrl() throws MalformedURLException {
		return new URL(url);
	}
	public void setUrl(URL url) {
		this.url = url.toString();
	}

	@Override
	public void validateRecord() throws InvalidDataException {
	}

	@Override
	public void mergeUpdate(Content newRecord) {
		if ( newRecord.name != null )
			name = newRecord.name;
		if ( newRecord.url != null )
			url = newRecord.url;
	}
}
