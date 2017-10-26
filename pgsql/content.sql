drop table if exists source_ref;
drop table if exists client_counter;
drop table if exists content_counter cascade;
drop table if exists content;
drop table if exists client;

create table if not exists source_ref (
    source_id varchar(32) not null,
    source_ref_int bigint not null default 0,
    primary key (source_id)
);

create table if not exists client (
    id uuid not null default uuid_generate_v1(),
    name text not null,
    primary key (id)
);

create table if not exists content (
    id uuid not null default uuid_generate_v1(),
    name text not null,
    url text not null,
    client_id uuid not null references client (id),
    primary key (id)
);

create unique index by_url on content (url);

--- Inherited table, ideally nothing should be inserted here
--- I define a primary key just in case
create table if not exists content_counter (
    content_id uuid not null references content (id),
    pageview_date date not null default now(),
    pageview_count bigint not null default 0,
    primary key (pageview_date, content_id)
);

create table if not exists client_counter (
    client_id uuid not null references client (id),
    pageview_date date not null default now(),
    pageview_count bigint not null default 0,
    primary key (pageview_date, client_id)
);


--- Source Reference procs
create or replace function get_source_ref ( sid varchar(32), out source_reference bigint ) as $$
    begin
        select source_ref_int into source_reference from source_ref where source_id = sid;
    end
$$ language plpgsql;

create or replace function set_source_ref ( sid varchar(32), inout source_reference bigint ) as $$
    begin
        IF source_reference IS NULL THEN
            delete from source_ref where source_id = sid;
        ELSE
            update source_ref set
                source_ref_int = source_reference
            where
                source_id = sid;

            if NOT FOUND THEN
                insert into source_ref (source_id, source_ref_int) values (sid, source_reference);
            end if;
        END IF;
    end
$$ language plpgsql;

create or replace function get_message_ref ( out source_reference bigint ) as $$
    begin
        select get_source_ref into source_reference from get_source_ref('content_pv_queue');
    end
$$ language plpgsql;

create or replace function set_message_ref ( sr bigint, out source_reference bigint ) as $$
    begin
        select set_source_ref into source_reference from set_source_ref('content_pv_queue', sr);
    end
$$ language plpgsql;

--- Client procs
create or replace function find_client( i text default null ) returns setof client as $$
    begin
        return query
        select
            *
        from
            client c 
        where
            c.id = coalesce(i::uuid,id)
        order by
            c.id desc;
    end
$$ language plpgsql;

create or replace function create_client( n text, out client_id text ) as $$
    declare
        ctable text;
    begin
        select (id) into client_id from client where name = $1;
        if not found then
            WITH inserted_rows AS (
                insert into client (name) values ($1) returning client.id
            ) select (id) into client_id from inserted_rows;

            ctable := format('create table content_counter_%s (primary key (pageview_date,content_id)) inherits (content_counter)', replace(client_id,'-',''));
            EXECUTE ctable;
        end if;
    end
$$ language plpgsql;

create or replace function update_client( inout client_id text, n text, replacement bool default false ) as $$
    begin
        if client_id is null then
            RAISE EXCEPTION 'client_id may not be null';
        end if;
            
        if replacement is null then
            RAISE EXCEPTION 'invalid value for "replacment"';
        end if;

        if replacement = TRUE then
            update client set
                name = n
            where
                id = client_id::uuid;
        else
            update client set
                name = COALESCE(n, name)
            where
                id = client_id::uuid;
        end if;

        if not found then
            RAISE EXCEPTION 'client_id % was not found', client_id;
        end if;
    end
$$ language plpgsql;

create or replace function delete_client( inout client_id text ) as $$
    declare
        ctable text;
    begin
        if client_id is null then
            RAISE EXCEPTION 'client_id may not be null';
        end if;
            
        delete from client where id = client_id::uuid;
        if not found then
            RAISE EXCEPTION 'client_id % was not found', client_id;
        else
            ctable := format('drop table content_counter_%s', replace(client_id,'-',''));
            EXECUTE ctable;
        end if;
    end
$$ language plpgsql;

--- Content Procs
create or replace function find_content (  c text, i text ) returns setof content as $$
    begin
        --- client_id is required
        if c is null then
            RAISE EXCEPTION 'content_id may not be null';
        end if;

        return query
        select
            *
        from
            content c
        where
            c.id = coalesce(i::uuid,c.id)
        and c.client_id = c::uuid
        order by
            c.id desc;
    end
$$ language plpgsql;

create or replace function create_content ( c uuid, n text, u text, out rv text ) as $$
    begin
        select
            (id) into rv
        from
            content
        where
            client_id = c::uuid
        and name = n
        and url = u;

        if not found then
            WITH inserted_rows AS (
                insert into content (client_id, name, url) values (c, n, u) returning content.*
            ) select (id) into rv from inserted_rows;
        end if;
    end
$$ language plpgsql;

create or replace function update_content( inout i text, n text, u text, replacement bool default false ) as $$
    begin
        if i is null then
            raise exception 'content_id may not be null';
        end if;

        if replacement = TRUE then
            update content set
                name  = n,
                url = u
            where
                id = i::uuid;
        else
            update content set
                name = coalesce(n,name),
                url = coalesce(u,name)
            where
                id = i::uuid;
        end if;

        if not found then
            raise exception 'content_id % was not found', i;
        end if;
    end
$$ language plpgsql;

create or replace function delete_content( inout i text ) as $$
    begin
        if i is null then
            RAISE EXCEPTION 'content_id may not be null';
        end if;

        delete from content where id = i::uuid;
        if not found then
            RAISE EXCEPTION 'content id % was not found', i;
        end if;
    end
$$ language plpgsql;

--- Pageview Procs
create or replace function client_content_pageview ( i uuid, pvd timestamp, out rv bigint ) as $$
    begin
        select client_pageview((select client_id from content where id = i), pvd) into rv;
    end
$$ language plpgsql;

create or replace function client_pageview ( c uuid, pvd timestamp, out rv bigint ) as $$
    begin
        with pv_count as (
            insert into client_counter as cc
                (client_id, pageview_date, pageview_count)
            values
                ($1,date($2),1)
            on conflict (pageview_date, client_id) do update set
                pageview_count = cc.pageview_count + 1
            returning cc.pageview_count
        ) select pageview_count into strict rv from pv_count;
    end
$$ language plpgsql;

create or replace function content_pageview ( i uuid, out rv bigint ) as $$
    begin
        with pv_count as (
            insert into content_counter as cc
                (content_id, pageview_count)
            values
                ($1,1)
            on conflict (pageview_date, content_id) do update set
                pageview_count = cc.pageview_count + 1
            returning cc.pageview_count
        ) select pageview_count into strict rv from pv_count;
    end
$$ language plpgsql;

create or replace function content_pageview ( i uuid, pvd timestamp, out rv bigint ) as $$
    declare
        client_id text;
    begin
        select replace(c.client_id::text,'-','') into client_id from content c where c.id = $1;
        select content_pageview(client_id, i, pvd) into rv;
    end
$$ language plpgsql;

create or replace function content_pageview ( c text, i uuid, pvd timestamp, out rv bigint ) as $$
    declare
        insert_statement text;
    begin
        ---
        --- Dynamic SQL is slow as there are no cached query plans
        --- if we're partitioning tables by customer id though, then read
        --- performance must be the goal
        ---
        insert_statement := format('with pv_count as (
                insert into content_counter_%s as cc
                    (content_id, pageview_date, pageview_count)
                values
                    (%s,%s,1)
                on conflict (pageview_date, content_id) do update set
                    pageview_count = cc.pageview_count + 1
                returning cc.pageview_count
            ) select pageview_count from pv_count',
            replace($1,'-',''),
            quote_literal(i),
            quote_literal(date(pvd)));

	EXECUTE insert_statement into rv;
    end
$$ language plpgsql;
