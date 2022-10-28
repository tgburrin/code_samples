package net.tgburrin.pageview_svc.client;

import java.util.UUID;

import org.springframework.data.jdbc.repository.query.Query;
import org.springframework.data.repository.CrudRepository;
import org.springframework.data.repository.query.Param;
import org.springframework.stereotype.Repository;

@Repository
public interface ClientRepository extends CrudRepository<Client, UUID> {
	@Query("select id, name from pageview.client where name=:name")
	Client findByName(@Param("name") String name);
}
