package net.tgburrin.pageview_svc.content;

import java.net.URL;
import java.util.List;
import java.util.UUID;

import org.springframework.data.jdbc.repository.query.Query;
import org.springframework.data.repository.CrudRepository;
import org.springframework.data.repository.query.Param;

public interface ContentRepository extends CrudRepository<Content, UUID> {
	@Query("select id, name, url, client_id from pageview.content where name=:name")
	List<Content> findByName(@Param("name") String name);

	@Query("select id, name, url, client_id from pageview.content where url=:url")
	Content findByUrl(@Param("url") URL url);
}
