#include <stdio.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <signal.h>

#include <libgen.h>

#include <time.h>
#include <fcntl.h>
#include <sys/time.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <pthread.h>

// https://www.postgresql.org/
#include <libpq-fe.h>

// https://github.com/edenhill/librdkafka
#include <librdkafka/rdkafka.h>

// https://github.com/DaveGamble/cJSON.git
#include <cjson/cJSON.h>

#define KAFKA_POLL_TIMEOUT_MS 10
#define STATS_TIMER_SEC 5

// General use
char *print_usage ( char * );
void error_exit( const char * );
void graceful_shutdown ();
void record_timer ();

// Configuration related
char *set_config_section_str ( cJSON *, char * );
int set_config_section_int ( cJSON *, char * );
void get_postgres_uri ( char * );

// Postgres related
void init_postgres ();
uint64_t get_last_source_ref ();
void set_last_source_ref ( uint64_t );
void update_pageview_count ( char * );

// Kafka related
void init_kafka ( uint64_t * );

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

static uint64_t uncommitted_count = 0;
static uint64_t batch_commit_size = 1000;
static volatile uint32_t uncommitted_timeout_sec = 10;
static struct timespec committed_time;
static uint64_t total_sent = 0;
static uint64_t total_committed = 0;

static volatile bool proc_running = false;
static volatile int exit_status = EXIT_SUCCESS;

static pthread_mutex_t counter_lck = PTHREAD_MUTEX_INITIALIZER;

void record_timer ( ) {
    uint64_t mycount_s = 0;
    uint64_t mycount_c = 0;

    while(proc_running) {

        pthread_mutex_lock(&counter_lck);
        uint64_t diff_s = total_sent - mycount_s;
        mycount_s = total_sent;

        uint64_t diff_c = total_committed - mycount_c;
        mycount_c = total_committed;
        pthread_mutex_unlock(&counter_lck);

        printf("%lf/s sent, %lf/s committed\n",
            (double)diff_s/(double)STATS_TIMER_SEC,
            (double)diff_c/(double)STATS_TIMER_SEC);

        sleep(STATS_TIMER_SEC);
    }

    pthread_exit(NULL);
}

void error_exit(const char *errstr) {
    fprintf(stderr, "ERROR: %s\n", errstr);

    if ( dbh )
        PQfinish(dbh);

    exit(EXIT_FAILURE);
}

void init_postgres () {
    char uri[1024];
    get_postgres_uri(uri);

    dbh = PQconnectdb(uri);

    if ( PQstatus(dbh) != CONNECTION_OK )
        error_exit(PQerrorMessage(dbh));
}

void start_transaction () {
    pgresult = PQexec(dbh, "BEGIN");
    if ( PQresultStatus(pgresult) != PGRES_COMMAND_OK ) {
        fprintf(stderr, "Could not begin transaction on postgres server\n");
        exit_status = EXIT_FAILURE;
        proc_running = false;
    }
    PQclear(pgresult);
}

void commit_transaction () {
    pgresult = PQexec(dbh, "COMMIT");
    if ( PQresultStatus(pgresult) != PGRES_COMMAND_OK ) {
        fprintf(stderr, "Could not begin transaction on postgres server\n");
        exit_status = EXIT_FAILURE;
        proc_running = false;
    }
    PQclear(pgresult);
}

uint64_t get_last_source_ref () {
    uint64_t rv;

    pgresult = PQexec(dbh, "select get_message_ref()");

    if ( PQresultStatus(pgresult) != PGRES_TUPLES_OK )
        error_exit(PQresultErrorMessage(pgresult));

    //fprintf(stdout, "%d rows returned\n", PQntuples(pgresult));

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

    if ( *source_ref != RD_KAFKA_OFFSET_BEGINNING ) {
        fprintf(stdout, "Finding last message we left off at: %"PRId64"\n", *source_ref);

        rd_kafka_message_t *msg;
        bool found_first_msg = false;
        while ( true ) {
            rd_kafka_poll(rk, 0);
            if ( (msg = rd_kafka_consume(msg_t, 0, KAFKA_POLL_TIMEOUT_MS)) ) {
                if ( msg->err && msg->err == RD_KAFKA_RESP_ERR__PARTITION_EOF )
                    break;

                if ( msg->len > 0 && msg->offset == *source_ref ) {
                    found_first_msg = true;
                    break;
                }
            }
        }

        if ( !found_first_msg ) {
            char err[1024];
            sprintf(err, "Could not find message with source ref %"PRId64, *source_ref);
            error_exit(err);
        } else {
            fprintf(stdout, "Located.  We didn't fall off of the queue\n");
        }
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
    char *message  = (char *)malloc(1024); // We can be sloppy about cleaning this up since it happens on exit()

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

    if ( (section = cJSON_GetObjectItem(root, "batch_commit_size")) != NULL && 
         section->type == cJSON_Number &&
         section->valueint > 0 )
        batch_commit_size = section->valueint;

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

    if ( root != NULL )
        cJSON_Delete(root);
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

void get_postgres_uri ( char *uri ) {
    strcpy(uri, "postgres://");

    if ( postgres_config.username != NULL ) {
        strcpy(uri+strlen(uri), postgres_config.username);

        if ( postgres_config.password != NULL )
            sprintf(uri+strlen(uri), ":%s", postgres_config.password);

        strcpy(uri+strlen(uri), "@");
    }

    if ( postgres_config.hostname != NULL ) {
        strcpy(uri+strlen(uri), postgres_config.hostname);

        if ( postgres_config.port != 0 && postgres_config.port > 0 )
            sprintf(uri+strlen(uri), ":%d", postgres_config.port);
    }

    if ( postgres_config.database != NULL )
        sprintf(uri+strlen(uri), "/%s", postgres_config.database);
}

void graceful_shutdown () {
    proc_running = false;
}

int main (int argc, char **argv) {
    rd_kafka_message_t *msg;
    uint64_t source_ref;

    static struct timespec now;

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
    clock_gettime(CLOCK_REALTIME, &now);

    // initialize the database connection
    init_postgres();

    // retrieve our source reference
    source_ref = get_last_source_ref();
    fprintf(stdout, "Starting from %"PRId64"\n", source_ref);

    // replay the queue from our source reference
    init_kafka(&source_ref);

    fprintf(stdout, "%s: %s\n", rd_kafka_name(rk), rd_kafka_memberid(rk));

    signal(SIGINT, graceful_shutdown);

    pthread_mutex_init(&counter_lck,NULL);

    pthread_t record_counter;
    pthread_create( &record_counter, NULL, (void *)(&record_timer),  NULL);

    proc_running = true;
    uint64_t lastoffset = 0;

    while( proc_running ) {
        rd_kafka_poll(rk, 0);

        clock_gettime(CLOCK_REALTIME, &now);

        if ( (msg = rd_kafka_consume(msg_t, 0, KAFKA_POLL_TIMEOUT_MS)) ) {
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
                //set_last_source_ref(msg->offset);
                fprintf(stdout, "End of queue reached for %s\n", rd_kafka_topic_name(msg->rkt));
                fprintf(stdout, "Consumed %"PRId64" messages\n", uncommitted_count);
            }

            if ( msg->len > 0 ) {
                lastoffset = msg->offset;

                if ( !(jsondoc = cJSON_Parse(msg->payload)) ) {
                    fprintf(stderr, "Can't parse msg: %s\n", (char *)msg->payload);
                    continue;
                }

                cJSON *msgtype = cJSON_GetObjectItem(jsondoc, "type");
                if ( msgtype == 0 || strcmp(msgtype->valuestring, "content_pageview") != 0 )
                    continue;

                cJSON *id = cJSON_GetObjectItem(jsondoc, "content_id");
                if ( id == 0 )
                    continue;

                if ( uncommitted_count == 0 ) {
                    clock_gettime(CLOCK_REALTIME, &committed_time);
                    start_transaction();
                }
                uncommitted_count++;

                update_pageview_count(id->valuestring);

                pthread_mutex_lock(&counter_lck);
                total_sent++;
                pthread_mutex_unlock(&counter_lck);

                cJSON_Delete(jsondoc);
            }

            rd_kafka_message_destroy(msg);
        }

        if ( uncommitted_count > 0 ) {
            bool do_commit = false;

            if ( uncommitted_count % batch_commit_size == 0 ) {
                fprintf(stdout, "Committing %"PRId64" based on batch\n", uncommitted_count);
                do_commit = true;
            } else if ( (now.tv_sec - committed_time.tv_sec) > uncommitted_timeout_sec ) {
                fprintf(stdout, "Committing %"PRId64" based on time\n", uncommitted_count);
                do_commit = true;
            }

            if ( do_commit ) {
                set_last_source_ref(lastoffset);
                commit_transaction();

                pthread_mutex_lock(&counter_lck);
                total_committed += uncommitted_count;
                pthread_mutex_unlock(&counter_lck);

                uncommitted_count = 0;
            }
        }
    }

    printf("Stopping consumer...\n");
    rd_kafka_consume_stop(msg_t, 0);

    printf("Removing topic...\n");
    rd_kafka_topic_destroy(msg_t);

    /* Some sort of bug here: this hangs indefinitely.
       It may be trying to store the offset on the server. */
    //printf("Closing connection and releasing resources...\n");
    //rd_kafka_destroy(rk);

    printf("Cleaning up Postgres...\n");
    if ( dbh )
        PQfinish(dbh);

    pthread_join(record_counter, NULL);
    pthread_mutex_destroy(&counter_lck);

    printf("Done\n");
    return exit_status;
}
