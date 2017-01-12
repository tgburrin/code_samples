# Makefile for Linux
CC = gcc -Wall
PROGS = kconsumer
CFLAGS = -std=gnu99 -pedantic
IFLAGS = -I.
SRCS = consumer.c

all: $(PROGS)

kconsumer: consumer.c
	$(CC) $(SRCS) $(IFLAGS) -o $(PROGS) $(CFLAGS) `pkg-config --libs --cflags rdkafka` `pkg-config --libs --cflags libcjson` `pkg-config --libs --cflags libpq` -lpthread

clean:
	rm -rf $(PROGS)
