package net.tgburrin.pageview_svc.content;

import java.net.MalformedURLException;
import java.net.URL;
import java.util.UUID;

import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Table;

@Table(name = "content", schema = "pageview")
public class Content {
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
}
