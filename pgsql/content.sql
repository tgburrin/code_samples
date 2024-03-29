drop table if exists pageview.source_ref;
drop table if exists pageview.client_counter;
drop table if exists pageview.content_counter cascade;
drop table if exists pageview.content;
drop table if exists pageview.client;

create table if not exists pageview.source_ref (
    source_id varchar(32) not null,
    source_ref_int bigint not null default 0,
    primary key (source_id)
);

create table if not exists pageview.client (
    id uuid not null default uuid_generate_v1(),
    name text not null,
    primary key (id)
);

create unique index by_name on pageview.client (name);

create table if not exists pageview.content (
    id uuid not null default uuid_generate_v1(),
    name text not null,
    url text not null,
    client_id uuid not null references client (id),
    primary key (id)
);

create unique index by_url on pageview.content (url);

--- Inherited table, ideally nothing should be inserted here
--- I define a primary key just in case
create table if not exists pageview.content_counter (
    content_id uuid not null references content (id),
    pageview_date date not null default now(),
    pageview_count bigint not null default 0,
    primary key (pageview_date, content_id)
);

create table if not exists pageview.client_counter (
    client_id uuid not null references client (id),
    pageview_date date not null default now(),
    pageview_count bigint not null default 0,
    primary key (pageview_date, client_id)
);

create or replace function pageview.get_source_ref ( sid varchar(32) ) returns bigint as $$
    declare
        rv bigint;
    begin
        select source_ref_int into rv from pageview.source_ref where source_id = sid;
        return rv;
    end
$$ language plpgsql;

create or replace function pageview.set_source_ref ( sid varchar(32), sr bigint ) returns bigint as $$
    begin
        IF sr IS NULL THEN
            delete from pageview.source_ref where source_id = sid;
        ELSE
            update pageview.source_ref set
                source_ref_int = sr
            where
                source_id = sid;

            if NOT FOUND THEN
                insert into pageview.source_ref (source_id, source_ref_int) values (sid, sr);
            end if;
        END IF;

        return sr;
    end
$$ language plpgsql;

create or replace function pageview.get_message_ref ( ) returns bigint as $$
    declare
        rv bigint;
    begin
        select get_source_ref into rv from pageview.get_source_ref('content_pv_queue');
        return rv;
    end
$$ language plpgsql;

create or replace function pageview.set_message_ref ( sr bigint ) returns bigint as $$
    declare
        rv bigint;

    begin
        select set_source_ref into rv from pageview.set_source_ref('content_pv_queue', sr);
        return rv;
    end
$$ language plpgsql;

create or replace function pageview.create_client( n text ) returns uuid as $$
    declare
        rv text;
        ctable text;
    begin
        select (id) into rv from pageview.client where name = $1;
        if not found then
            WITH inserted_rows AS (
                insert into pageview.client (name) values ($1) returning client.id
            ) select (id) into rv from inserted_rows;

            ctable := format('create table pageview.content_counter_%s (primary key (pageview_date,content_id)) inherits (content_counter)', replace(rv,'-',''));
            EXECUTE ctable;
        end if;
        return rv;
    end
$$ language plpgsql;

create or replace function pageview.content_add ( client_id uuid, n text, u text ) returns uuid as $$
    declare
        rv text;
    begin
        select (id) into rv from pageview.content where url = $3;

        if not found then
            WITH inserted_rows AS (
                insert into pageview.content (client_id, name, url) values ($1, $2, $3) returning content.*
            ) select (id) into rv from inserted_rows;
        end if;

        return rv;
    end
$$ language plpgsql;

create or replace function pageview.content_get ( u text ) returns uuid as $$
    declare
        rv text;
    begin
        select id into strict rv from pageview.content where url = $1;
        return rv;
        EXCEPTION
            WHEN NO_DATA_FOUND THEN
                RAISE EXCEPTION 'url % not found', $1;
            WHEN TOO_MANY_ROWS THEN
                RAISE EXCEPTION 'url % not unique', $1;
    end
$$ language plpgsql;

create or replace function pageview.client_content_pageview ( i uuid, pvd timestamp ) returns bigint as $$
    declare
        rv bigint;
    begin
        select pageview.client_pageview((select client_id from pageview.content where id = i), pvd) into rv;
        return rv;
    end
$$ language plpgsql;

create or replace function pageview.client_pageview ( c uuid, pvd timestamp ) returns bigint as $$
    declare
        rv bigint;
    begin
        with pv_count as (
            insert into pageview.client_counter as cc
                (client_id, pageview_date, pageview_count)
            values
                ($1,date($2),1)
            on conflict (pageview_date, client_id) do update set
                pageview_count = cc.pageview_count + 1
            returning cc.pageview_count
        ) select pageview_count into strict rv from pv_count;
        return rv;
    end
$$ language plpgsql;

create or replace function pageview.content_pageview ( i uuid ) returns bigint as $$
    declare
        rv bigint;
    begin
        with pv_count as (
            insert into pageview.content_counter as cc
                (content_id, pageview_count)
            values
                ($1,1)
            on conflict (pageview_date, content_id) do update set
                pageview_count = cc.pageview_count + 1
            returning cc.pageview_count
        ) select pageview_count into strict rv from pv_count;
        return rv;
    end
$$ language plpgsql;

create or replace function pageview.content_pageview ( i uuid, pvd timestamp ) returns bigint as $$
    declare
        rv bigint;
        client_id text;
    begin
        select replace(c.client_id::text,'-','') into client_id from pageview.content c where c.id = $1;
        select pageview.content_pageview(client_id, i, pvd) into rv;
        return rv;
    end
$$ language plpgsql;

create or replace function pageview.content_pageview ( c text, i uuid, pvd timestamp ) returns bigint as $$
    declare
        rv bigint;
        insert_statement text;
    begin
        ---
        --- Dynamic SQL is slow as there are no cached query plans
        --- if we're partitioning tables by customer id though, then read
        --- performance must be the goal
        ---
        insert_statement := format('with pv_count as (
                insert into pageview.content_counter_%s as cc
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
        return rv;
    end
$$ language plpgsql;

