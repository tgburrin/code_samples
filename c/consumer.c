#include <stdio.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>

#include <time.h>
#include <sys/time.h>

// https://www.postgresql.org/
#include <libpq-fe.h>

// https://github.com/edenhill/librdkafka
#include <librdkafka/rdkafka.h>

// https://github.com/DaveGamble/cJSON.git
#include <cjson/cJSON.h>

#define BROKER_LIST "lofr.tgburrin.net:9092"
#define POSTGRES_URI "postgresql://tgburrin:junk_password@db-pgsql.tgburrin.net/tgburrin"

// comma separated list of brokers
#define TOPIC "messages"
#define CONSUMER_GROUP "content_pageview_processor"

static rd_kafka_t *rk;
static rd_kafka_topic_t *msg_t;

static cJSON *jsondoc;

static PGconn *dbh;
static PGresult *pgresult;

static uint64_t committed_count = 0;
static struct timespec committed_time;

void error_exit(const char *errstr) {
    fprintf(stderr, "ERROR: %s\n", errstr);

    if ( dbh )
        PQfinish(dbh);

    exit(1);
}

void init_pq () {
    dbh = PQconnectdb(POSTGRES_URI);

    if ( PQstatus(dbh) != CONNECTION_OK )
        error_exit(PQerrorMessage(dbh));
}

uint64_t get_last_source_ref () {
    uint64_t rv;

    pgresult = PQexec(dbh, "select get_message_ref()");

    if ( PQresultStatus(pgresult) != PGRES_TUPLES_OK )
        error_exit(PQresultErrorMessage(pgresult));

    fprintf(stdout, "%d rows returned\n", PQntuples(pgresult));

    if ( PQgetisnull(pgresult, 0, 0) )
        return RD_KAFKA_OFFSET_BEGINNING;
    else if ( sscanf(PQgetvalue(pgresult, 0, 0), "%"SCNd64, &rv) == 0 )
        error_exit("Could not scan source ref");

    PQclear(pgresult);
    return rv;
}

void init_k (uint64_t *source_ref) {
    char errstr[1024] = "\0";

    rd_kafka_conf_t *conf = 0;
    rd_kafka_topic_conf_t *topic_conf = 0;

    conf = rd_kafka_conf_new();
    topic_conf = rd_kafka_topic_conf_new();

    if ( rd_kafka_conf_set(conf, "bootstrap.servers", BROKER_LIST, errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    if ( rd_kafka_conf_set(conf, "enable.auto.offset.store", "false", errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    if ( rd_kafka_conf_set(conf, "offset.store.method", "none", errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    if ( rd_kafka_conf_set(conf, "group.id", CONSUMER_GROUP, errstr, sizeof(errstr)) !=
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    //if ( rd_kafka_conf_set(conf, "debug", "all", errstr, sizeof(errstr)) !=
    //     RD_KAFKA_CONF_OK )
    //    error_exit(errstr);

    if ( rd_kafka_topic_conf_set(topic_conf, "auto.offset.reset", "beginning", errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    if ( rd_kafka_topic_conf_set(topic_conf, "auto.commit.enable", "false", errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    fprintf(stdout, "Creating new consumer\n");
    if ( !(rk = rd_kafka_new(RD_KAFKA_CONSUMER, conf, errstr, sizeof(errstr))) )
        error_exit(errstr);

    if (rd_kafka_brokers_add(rk, BROKER_LIST) == 0)
        error_exit("No valid brokers specified");

    msg_t = rd_kafka_topic_new(rk, TOPIC, topic_conf);

    fprintf(stdout, "Starting consumer for topic '%s'\n", TOPIC);
    //if ( rd_kafka_consume_start(msg_t, 0, RD_KAFKA_OFFSET_BEGINNING) == -1 ) {
    if ( rd_kafka_consume_start(msg_t, 0, *source_ref) == -1 ) {
        rd_kafka_resp_err_t err = rd_kafka_last_error();
        error_exit(rd_kafka_err2str(err));
    }
}

void set_last_source_ref ( uint64_t ref ) {
    const char *paramValues[1];

    char numval[64] = "\0";

    sprintf(numval, "%"PRId64, ref);
    paramValues[0] = numval;

    pgresult = PQexecParams(dbh,
                            "select set_message_ref($1::bigint)",
                            1,
                            NULL,
                            paramValues,
                            NULL,
                            NULL,
                            1);

    if ( PQresultStatus(pgresult) != PGRES_TUPLES_OK )
        fprintf(stderr, "Could not update source ref to %"PRId64"\n", ref);

    PQclear(pgresult);
}

void update_pageview_count ( char *id ) {
    const char *paramValues[1];

    char ide[64] = "\0";
    PQescapeStringConn(dbh, ide, id, sizeof(ide), NULL);

    paramValues[0] = ide;
    pgresult = PQexecParams(dbh,
                            "select content_pageview($1::uuid)",
                            1,
                            NULL,
                            paramValues,
                            NULL,
                            NULL,
                            1);

    if ( PQresultStatus(pgresult) != PGRES_TUPLES_OK )
        fprintf(stderr, "Could not process pageview event for %s: %s\n", id, PQerrorMessage(dbh));

    PQclear(pgresult);
}

int main (int arc, char **argv) {
    rd_kafka_message_t *msg;
    int counter = 0;
    uint64_t source_ref;

    //fprintf(stdout, "Debugging options are: %s\n", RD_KAFKA_DEBUG_CONTEXTS);
    //rd_kafka_conf_properties_show(stdout);
    //exit(0);

    // record the time that we started
    clock_gettime(CLOCK_REALTIME, &committed_time);

    // initialize the database connection
    init_pq();

    // retrieve our source reference
    source_ref = get_last_source_ref();
    fprintf(stdout, "Starting from %"PRId64"\n", source_ref);

    // replay the queue from our source reference
    init_k(&source_ref);

    fprintf(stdout, "%s: %s\n", rd_kafka_name(rk), rd_kafka_memberid(rk));

    while( true ) {
        rd_kafka_poll(rk, 0);

        if ( (msg = rd_kafka_consume(msg_t, 0, 250)) ) {
            if ( msg->err && msg->err != RD_KAFKA_RESP_ERR__PARTITION_EOF ) {
                if ( msg->rkt )
                    fprintf(stderr, "%s: %s at offset %"PRId64"\n",
                            rd_kafka_topic_name(msg->rkt),
                            rd_kafka_message_errstr(msg),
                            msg->offset);
                else
                    fprintf(stderr, "Consume error: %s: %s", rd_kafka_err2str(msg->err),
                                                             rd_kafka_message_errstr(msg));
            } else if ( msg->err && msg->err == RD_KAFKA_RESP_ERR__PARTITION_EOF ) {
                set_last_source_ref(msg->offset);
                fprintf(stdout, "End of queue reached for %s at offset %"PRId64"\n",
                                    rd_kafka_topic_name(msg->rkt),
                                    msg->offset);
                fprintf(stdout, "Consumed %d messages\n", counter);
                counter = 0;
            }

            if ( msg->len > 0 ) {
                counter++;

                //fprintf(stdout, "Message for %s of %"PRId32" bytes at offset %"PRId64"\n",
                //                                                         rd_kafka_topic_name(msg->rkt),
                //                                                         msg->len,
                //                                                         msg->offset);

                if ( !(jsondoc = cJSON_Parse(msg->payload)) ) {
                    fprintf(stderr, "Can't parse msg: %s\n", (char *)msg->payload);
                    continue;
                }

                cJSON *msgtype = cJSON_GetObjectItem(jsondoc, "type");
                if ( msgtype == 0 || strcmp(msgtype->valuestring, "content_pageview") != 0 )
                    continue;

                cJSON *id = cJSON_GetObjectItem(jsondoc, "id");
                if ( id == 0 )
                    continue;
                //printf("ID: %s (%"PRId64")\n", (char *)id->valuestring, msg->offset);

                update_pageview_count(id->valuestring);

                // commit & update source ref

                cJSON_Delete(jsondoc);
            }

            rd_kafka_message_destroy(msg);
        } else {
            //printf("Polling...\n");
        }
    }

    printf("Doing cleanup...\n");

    rd_kafka_consume_stop(msg_t, RD_KAFKA_PARTITION_UA);
    rd_kafka_topic_destroy(msg_t);
    rd_kafka_destroy(rk);

    if ( dbh )
        PQfinish(dbh);

    printf("Done\n");
    return 0;
}
