#include <stdio.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>

#include <libgen.h>

#include <time.h>
#include <fcntl.h>
#include <sys/time.h>
#include <sys/stat.h>
#include <sys/types.h>

// https://www.postgresql.org/
#include <libpq-fe.h>

// https://github.com/edenhill/librdkafka
#include <librdkafka/rdkafka.h>

// https://github.com/DaveGamble/cJSON.git
#include <cjson/cJSON.h>

// General use
char *print_usage ( char * );
void error_exit( const char * );

// Configuration related
char *set_config_section_str ( cJSON *, char * );
int set_config_section_int ( cJSON *, char * );
char *get_postgres_uri ( );

void init_postgres ( char * );
uint64_t get_last_source_ref ();
void set_last_source_ref ( uint64_t );
void update_pageview_count ( char * );

void init_kafka ( uint64_t );

struct KafkaConfig {
    char broker_list[2048];
    char *topic;
    char consumer_group[1024];
    bool debug;
} kafka_config;

struct PostgresConfig {
    char *username;
    char *password;
    char *hostname;
    int  port;
    char *database;
    char **options;
} postgres_config;

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

    exit(EXIT_FAILURE);
}

void init_postgres (char *uri) {
    dbh = PQconnectdb(uri);

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

void init_kafka (uint64_t *source_ref) {
    char errstr[1024] = "\0";

    rd_kafka_conf_t *conf = 0;
    rd_kafka_topic_conf_t *topic_conf = 0;

    conf = rd_kafka_conf_new();
    topic_conf = rd_kafka_topic_conf_new();

    if ( rd_kafka_conf_set(conf, "bootstrap.servers", kafka_config.broker_list, errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    if ( rd_kafka_conf_set(conf, "group.id", kafka_config.consumer_group, errstr, sizeof(errstr)) !=
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    // Hardcoding all of this so that the client can keep track instead
    if ( rd_kafka_conf_set(conf, "enable.auto.offset.store", "false", errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    if ( rd_kafka_conf_set(conf, "offset.store.method", "none", errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    if ( rd_kafka_topic_conf_set(topic_conf, "auto.offset.reset", "beginning", errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    if ( rd_kafka_topic_conf_set(topic_conf, "auto.commit.enable", "false", errstr, sizeof(errstr)) != 
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    if ( kafka_config.debug && rd_kafka_conf_set(conf, "debug", "all", errstr, sizeof(errstr)) !=
         RD_KAFKA_CONF_OK )
        error_exit(errstr);

    fprintf(stdout, "Creating new consumer\n");
    if ( !(rk = rd_kafka_new(RD_KAFKA_CONSUMER, conf, errstr, sizeof(errstr))) )
        error_exit(errstr);

    if (rd_kafka_brokers_add(rk, kafka_config.broker_list) == 0)
        error_exit("No valid brokers specified");

    msg_t = rd_kafka_topic_new(rk, kafka_config.topic, topic_conf);

    fprintf(stdout, "Starting consumer for topic '%s'\n", kafka_config.topic);
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

char *print_usage( char *program_name ) {
    char message[1024];
    sprintf(message, "Usage: %s <options>\n", program_name);
    sprintf(message + strlen(message), "\t-c <config file> (required)\n");
    sprintf(message + strlen(message), "\t-h this help menu\n");

    return message;
}

void parse_config ( char *config_file ) {
    struct stat statbuf;
    int fd;

    strcpy(kafka_config.consumer_group, "content_pageview_processor");
    kafka_config.debug = false;

    if ( stat(config_file, &statbuf) < 0 ) {
        fprintf(stderr, "Could not stat %s: %s\n", config_file, strerror(errno));
        exit(EXIT_FAILURE);
    }

    char file_content[statbuf.st_size+1];

    if ( (fd = open(config_file, 0)) < 0 ) {
        fprintf(stderr, "Could open %s: %s\n", config_file, strerror(errno));
        exit(EXIT_FAILURE);
    }

    int bytes_read = 0;
    while ( bytes_read < statbuf.st_size ) {
        int i = read(fd, file_content+bytes_read, statbuf.st_size - bytes_read);
        if ( i < 0 ) {
            fprintf(stderr, "Error while reading %s: %s\n", config_file, strerror(errno));
            exit(EXIT_FAILURE);
        }
        bytes_read += i;
    }
    file_content[bytes_read+1] = '\0';

    cJSON *root = 0;
    if ( (root = cJSON_Parse(file_content)) == NULL ) {
        fprintf(stderr, "Could not parse %s as json\n", config_file);
        exit(EXIT_FAILURE);
    }

    cJSON *section;

    // Pageview details
    if ( (section = cJSON_GetObjectItem(root, "pageview_connection")) == NULL ) {
        fprintf(stderr, "Could not locate pageview_connection section in: %s\n", config_file);
        exit(EXIT_FAILURE);
    } else {
        cJSON *item;
        if ( (item = cJSON_GetObjectItem(section, "brokers")) == NULL ) {
            fprintf(stderr, "Could not locate pageview_connection->brokers section in: %s\n", config_file);
            exit(EXIT_FAILURE);
        }

        switch ((item->type) & 0xFF) {
            case cJSON_Array:
                for ( int i = 0; i < cJSON_GetArraySize(item); i++ ) {
                    cJSON *broker = cJSON_GetArrayItem(item, i);
                    if ( broker == NULL || broker->type != cJSON_String ) {
                        fprintf(stderr, "Invalid broker value at index %d\n", i);
                        exit(EXIT_FAILURE);
                    } else {
                        // This will waste one byte in the beginning, but will account for the , and \0 going forward
                        if ( sizeof(kafka_config.broker_list) - 
                            (strlen(kafka_config.broker_list) + strlen(broker->valuestring) + 2) < 0) {
                            fprintf(stderr, "Broker list is too long, skipping %s\n", broker->valuestring);
                            continue;
                        }

                        if ( i > 0 )
                            strcpy(kafka_config.broker_list + strlen(kafka_config.broker_list), ",");

                        strcpy(kafka_config.broker_list + strlen(kafka_config.broker_list), broker->valuestring);
                    }
                }
                break;
            case cJSON_String:
                strcpy(kafka_config.broker_list, item->valuestring);
                break;
            default:
                fprintf(stderr, "pageview_connection->brokers must be an array of strings or a string\n");
                exit(EXIT_FAILURE);
        }

        if ( ( kafka_config.topic = set_config_section_str(section, "topic") ) == NULL ) {
            fprintf(stderr, "Invalid topic provided\n");
            exit(EXIT_FAILURE);
        }

        if ( (item = cJSON_GetObjectItem(section, "debug")) != NULL && item->type == cJSON_True )
            kafka_config.debug = true;
    }

    // content connection
    if ( (section = cJSON_GetObjectItem(root, "content_connection")) == NULL ) {
        fprintf(stderr, "Could not locate content_connection in: %s\n", config_file);
        exit(EXIT_FAILURE);
    } else {
        postgres_config.username = set_config_section_str(section, "username");
        postgres_config.password = set_config_section_str(section, "password");
        postgres_config.hostname = set_config_section_str(section, "hostname");
        postgres_config.database = set_config_section_str(section, "database");
        postgres_config.port     = set_config_section_int(section, "port");
    }
}

char *set_config_section_str ( cJSON *base, char *section ) {
    cJSON *item;
    char *rv = 0;

    if ( (item = cJSON_GetObjectItem(base, section)) != NULL && item->type != cJSON_String ) {
        fprintf(stderr, "%s->%s must be a string if provided\n", base->string, section);
        exit(EXIT_FAILURE);
    } else if ( (item = cJSON_GetObjectItem(base, section)) != NULL ) {
        rv = (char *)malloc(strlen(item->valuestring)+1);
        strcpy(rv, item->valuestring);
    }

    return rv;
}

int set_config_section_int ( cJSON *base, char *section ) {
    cJSON *item;
    int rv = 0;

    if ( (item = cJSON_GetObjectItem(base, section)) != NULL && item->type != cJSON_Number ) {
        fprintf(stderr, "%s->%s must be an int if provided\n", base->string, section);
        exit(EXIT_FAILURE);
    } else if ( (item = cJSON_GetObjectItem(base, section)) != NULL ) {
        rv = item->valueint;
    }

    return rv;
}

char *get_postgres_uri () {
    char *uri = (char *)malloc(1024);
    strcpy(uri, "postgres://");

    if ( postgres_config.username != NULL ) {
        strcpy(uri+strlen(uri), postgres_config.username);

        if ( postgres_config.password != NULL )
            sprintf(uri+strlen(uri), ":%s", postgres_config.password);

        strcpy(uri+strlen(uri), "@");
    }

    if ( postgres_config.hostname != NULL ) {
        strcpy(uri+strlen(uri), postgres_config.hostname);

        if ( postgres_config.port != NULL && postgres_config.port > 0 )
            sprintf(uri+strlen(uri), ":%d", postgres_config.port);
    }

    if ( postgres_config.database != NULL )
        sprintf(uri+strlen(uri), "/%s", postgres_config.database);

    return uri;
}

int main (int argc, char **argv) {
    rd_kafka_message_t *msg;
    int counter = 0;
    uint64_t source_ref;

    char *config_file = 0;

    int opt;
    extern char *optarg;
    extern int optind, opterr, optopt;

    while ((opt = getopt(argc, argv, "hc:k")) != -1) {
        switch (opt) {
            case 'c':
                config_file = optarg;
                break;
            case 'k':
                fprintf(stdout, "Debugging options are: %s\n", RD_KAFKA_DEBUG_CONTEXTS);
                rd_kafka_conf_properties_show(stdout);
                exit(EXIT_SUCCESS);
            case 'h':
                fprintf(stdout, "%s", print_usage(basename(argv[0])));
                exit(EXIT_SUCCESS);
            default:
                fprintf(stderr, "Unknown option provided\n%s", print_usage(basename(argv[0])));
                exit(EXIT_FAILURE);
        }
    }

    if ( config_file == NULL ) {
        fprintf(stderr, "-c is a required option\n%s", print_usage(basename(argv[0])));
        exit(EXIT_FAILURE);
    }

    parse_config(config_file);

    // record the time that we started
    clock_gettime(CLOCK_REALTIME, &committed_time);

    // initialize the database connection
    init_postgres(get_postgres_uri());

    // retrieve our source reference
    source_ref = get_last_source_ref();
    fprintf(stdout, "Starting from %"PRId64"\n", source_ref);

    // replay the queue from our source reference
    init_kafka(&source_ref);

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
