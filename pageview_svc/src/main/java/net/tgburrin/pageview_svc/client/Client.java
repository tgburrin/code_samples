package net.tgburrin.pageview_svc.client;

import java.util.UUID;

import org.springframework.data.annotation.Id;
import org.springframework.data.relational.core.mapping.Table;

@Table(value = "client")
public class Client {
	@Id
	private UUID id;
	private String name;
}
