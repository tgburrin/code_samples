# code_samples

These are several personal coding samples encompasing a single project.

#### Postgresql
This is the schema for a sample database that contains a very simple set of tables and a few stored procedures to go with them.
The main purpose of these tables are to manage some basic pieces of content, counters about them, and a source ref for a kafka queue (this is for recovery purposes in the event that the database is restored from a backup).  Interesting qualities of this schema include inheritied tables created by the 'create_client' sproc and the choice to use dynamic sql (in 'content_pageview') to partition the content pageview data by client.

#### Golang
This is the rest layer that abstracts the pgsql database and adds events to the kafka queue.

#### C
(deprecated) This is a simple consumer to read pageview events off of the kafka queue and executes a stored procedure for a subset of them.  This has been supplanted by the C++ project below.

#### CPP
This is a C++ version of the Kafka consumer which creates two threads, one for each stat type, and calls stored procedures depending on their purpose.  These two consumer threads both react to the content_pageview message type and keep their own copies of where they left off on the Kafka queue.

#### Java
This is a simple, fairly hardcoded client for generating pageview events against the rest layer.  Each layer except for this one is configurable by a json file that is included in the same directory.

#### Ansible
This is a set of ansible playbooks which will build (but not start) the component parts to either EC2 (where it builds the hosts) or local lan hosts.  In the case of EC2 it makes the assumption that you are on a LAN connected to a VPC via a VPN or something similar.
This repository is where you can find details about the dependencies for the sub projects above.
