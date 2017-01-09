create table source_ref (
    source_id varchar(32) not null,
    source_ref_int bigint not null default 0,
    primary key (source_id)
);

create table content (
    id uuid not null default uuid_generate_v1(),
    name text not null,
    url text not null,
    primary key (id)
);

create unique index by_url on content (url);

create table content_counter (
    id uuid not null references content (id),
    pageview_date date not null default now(),
    pageview_count bigint not null default 0,
    primary key (pageview_date, id)
);

create or replace function get_message_ref ( ) returns bigint as $$
    declare
        rv bigint;
    begin
        select source_ref_int into rv from source_ref where source_id = 'message_queue';
        return rv;
    end
$$ language plpgsql;

create or replace function set_message_ref ( sr bigint ) returns bigint as $$
    begin
        update source_ref set
            source_ref_int = sr
        where
            source_id = 'message_queue';

        if NOT FOUND THEN
            insert into source_ref (source_id, source_ref_int) values ( 'message_queue', sr );
        end if;
        return sr;
    end
$$ language plpgsql;

create or replace function content_add ( n text, u text ) returns uuid as $$
    declare
        rv text;
    begin
        select (id) into rv from content where url = $2;

        if not found then
            WITH inserted_rows AS (
                insert into content (name, url) values ($1, $2) returning content.*
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

create or replace function content_pageview ( i uuid ) returns bigint as $$
    declare
        rv bigint;
    begin
        with pv_count as (
            insert into content_counter as cc
                (id, pageview_count)
            values
                ($1,1)
            on conflict (pageview_date, id) do update set
                pageview_count = cc.pageview_count + 1
            returning cc.*
        ) select pageview_count into strict rv from pv_count;
        return rv;
    end
$$ language plpgsql;
