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

create or replace function get_message_ref ( ) returns bigint as $$
    declare
        rv bigint;
    begin
        select source_ref_int into rv from source_ref where source_id = 'content_pv_queue';
        return rv;
    end
$$ language plpgsql;

create or replace function set_message_ref ( sr bigint ) returns bigint as $$
    begin
        update source_ref set
            source_ref_int = sr
        where
            source_id = 'content_pv_queue';

        if NOT FOUND THEN
            insert into source_ref (source_id, source_ref_int) values ( 'content_pv_queue', sr );
        end if;
        return sr;
    end
$$ language plpgsql;

create or replace function create_client( n text ) returns uuid as $$
    declare
        rv text;
        ctable text;
    begin
        select (id) into rv from client where name = $1;
        if not found then
            WITH inserted_rows AS (
                insert into client (name) values ($1) returning client.id
            ) select (id) into rv from inserted_rows;

            ctable := format('create table content_counter_%s (primary key (pageview_date,content_id)) inherits (content_counter)', replace(rv,'-',''));
            EXECUTE ctable;
        end if;
        return rv;
    end
$$ language plpgsql;

create or replace function content_add ( client_id uuid, n text, u text ) returns uuid as $$
    declare
        rv text;
    begin
        select (id) into rv from content where url = $3;

        if not found then
            WITH inserted_rows AS (
                insert into content (client_id, name, url) values ($1, $2, $3) returning content.*
            ) select (id) into rv from inserted_rows;
        end if;

        return rv;
    end
$$ language plpgsql;

create or replace function content_get ( u text ) returns uuid as $$
    declare
        rv text;
    begin
        select id into strict rv from content where url = $1;
        return rv;
        EXCEPTION
            WHEN NO_DATA_FOUND THEN
                RAISE EXCEPTION 'url % not found', $1;
            WHEN TOO_MANY_ROWS THEN
                RAISE EXCEPTION 'url % not unique', $1;
    end
$$ language plpgsql;

create or replace function client_content_pageview ( i uuid, pvd timestamp ) returns bigint as $$
    declare
        rv bigint;
    begin
        select client_pageview((select client_id from content where id = i), pvd) into rv;
        return rv;
    end
$$ language plpgsql;

create or replace function client_pageview ( c uuid, pvd timestamp ) returns bigint as $$
    declare
        rv bigint;
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
        return rv;
    end
$$ language plpgsql;

create or replace function content_pageview ( i uuid ) returns bigint as $$
    declare
        rv bigint;
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
        return rv;
    end
$$ language plpgsql;

create or replace function content_pageview ( i uuid, pvd timestamp ) returns bigint as $$
    declare
        rv bigint;
        client_id text;
    begin
        select replace(c.client_id::text,'-','') into client_id from content c where c.id = $1;
        select content_pageview(client_id, i, pvd) into rv;
        return rv;
    end
$$ language plpgsql;

create or replace function content_pageview ( c text, i uuid, pvd timestamp ) returns bigint as $$
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
        return rv;
    end
$$ language plpgsql;

