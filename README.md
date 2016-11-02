# autofact
> Write: _the product is thoroughly pizzled._
>
>   -Philip K. Dick,  _Autofac_

Communications between the server and client are via websockets and most messages are serialized using [flatbuffers.](https://google.github.io/flatbuffers/)  An inventory of clients is persisted using [boltdb.](https://github.com/boltdb/bolt)

This is a work in progress.

At this point in time, nothing is encrypted and `ws` is being used so definitely don't use this where communications will go over public networks.

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
Autofactory is the server.  By default, it listens on `:8675` and processes incoming messages, sending responses to the client as appropriate.  Generally, this means printing out what the client sent.  In the future it will probably do something with the received messages.

When a client connects, it responds with either the client's ID, if it is a new client, or a welcome back message.  It also sends the client information on how it should behave.

This component is not necessary if autofact is running in serverless mode.
## autofact
Autofact runs on client nodes; it can also be run in serverless mode.
