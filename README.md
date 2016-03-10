# autofact
> Write: _the product is thoroughly pizzled._
>
>   -Philip K. Dick,  _Autofac_

This is an experiment to get an understanding of some things: mainly websockets and flatbuffers.

Communications between the server and client are via websockets and most messages are serialized using [flatbuffers.](https://google.github.io/flatbuffers/)  An inventory of clients is persisted using [boltdb.](https://github.com/boltdb/bolt)

This is a work in progress and should only be used for experimental purposes.

Things may change as I try things out.

At this point in time, nothing is encrypted and `ws` is being used so definitely don't use this where communcations will go over public networks.

Currently, only Linux systems are supported and this has only been tested on Debian Jessie.

## Dependencies
[InfluxDB](https://influxdata.com/) is used to store the facts.  Currently, it is assumed that InfluxDB is on the localhost and listening on port `8086`.

To install follow the instructions: https://influxdata.com/downloads/.

Once installed, the database and database user need to be created:

```
$ influx
Visit https://enterprise.influxdata.com to register for updates, InfluxDB server management, and monitoring.
Connected to http://localhost:8086 version 0.10.2
InfluxDB shell 0.10.2
> create user autoadmin with password 'thisisnotapassword'
> grant all privileges to autoadmin
> create database autofacts
> quit
```

Use the graph and dashboard builder of your choice: [Grafana](http://grafana.org/) is one option.

## autofactory
Autoctory is the server.  By default, it listens on `:8675` and processes incoming messages, sending responses to the client as appropriate.  Generally, this means printing out what the client sent.  In the future it will probably do something with the received messages.

When a client connects, it responds with either the client's ID, if it is a new client, or a welcome back message.  It also sends the client information on how it should behave.

## autofact-client
Autofact-client runs on client nodes.  It has a basic configuration, in `JSON`, that tells it where the server is.  If the client has already connected with the server, it also knows its own ID.

A client is responsible for maintaining the connection with the server, it does this by either sending messages, as configured, or, if a message hasn't been send in a certain period of time, sending a `ping` to the server.  If the client detects it has lost the connection with the server, it will try to re-establish the connection until for a certain amount of time before shutting down.  While disconnected, the client will continue gathering data and buffer it.  Once the connection is re-established, the buffered data will be sent to the server.

Currently, the client periodically gathers its CPU and Memory info and sends all pending messages back to the server after a pre-defined interval of time has been passed.

A client does not maintain any information about how it should operate, the server pushes this information to the client.

Most messages are sent as binary messages with the message payload being a bunch of bytes serialized with flatbuffers.

## Notes
The ClientID is used as the Client Name.  In the future the client's `hostname` should be used.

The Region is hardcoded as region isn't yet implemented.

## TODO

* Persist buffered data on the client side until the data has been sent.
* Add message id to the ack message.
* Track message sent vs ack'd.
* Use `hostname` as the Client's name.
* Add Region support.
