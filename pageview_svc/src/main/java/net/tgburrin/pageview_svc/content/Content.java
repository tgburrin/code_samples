package net.tgburrin.pageview_svc.content;

import java.util.UUID;

import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Table;

@Table(value = "content")
public class Content {
	@Id
	private UUID id;

	private String name;
	private String url;

}
