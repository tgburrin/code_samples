package net.tgburrin.pageview_svc.pageview;

import java.time.Instant;
import java.util.UUID;

import com.fasterxml.jackson.annotation.JsonProperty;

public class Pageview {
	private UUID contentId;
	private static final String type = "content_pageview";
	private Instant pageviewDt;

	public Pageview(UUID cid) {
		this.contentId = cid;
		this.pageviewDt = Instant.now();
	}

	@JsonProperty("content_id")
	public UUID getContentId() {
		return contentId;
	}
	public String getType() {
		return type;
	}

	@JsonProperty("pageview_dt")
	public Instant getPageviewDt() {
		return pageviewDt;
	}
}
