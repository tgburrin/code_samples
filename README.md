# code_samples

These are several personal coding samples encompasing a single project.

#### Postgresql
This is the schema for a sample database that contains a very simple set of tables and a few stored procedures to go with them.
The main purpose of these tables are to manage some basic pieces of content, counters about them, and a source ref for a kafka queue (this is for recovery purposes in the event that the database is restored from a backup).

#### Golang
This is the rest layer that abstracts the pgsql database and adds events to the kafka queue.

#### C
This is a simple consumer to read pageview events off of the kafka queue and executes a stored procedure for a subset of them.

#### Java
This is a simple, fairly hardcoded client for generating pageview events against the rest layer.  Each layer except for this one is configurable by a json file that is included in the same directory.
