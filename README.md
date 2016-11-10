# autofact
> Write: _the product is thoroughly pizzled._
>
>   -Philip K. Dick,  _Autofac_

Autofact periodically collects data about a client for basic monitoring purposes. Autofact is simple, fast, and uses minimal client resources. It can either collect the data locally as JSON or push it to a remote location using Websockets, usually [autofactory](https://github.com/mohae/autofact/tree/master/cmd/autofactory).

## About
[autofactory](https://github.com/mohae/autofact/tree/master/cmd/autofactory). Autofact's goal is to collect information about a client's usage with minimal impact on the client on which it is running. To accomplish this, Autofact uses [joefriday](https://github.com/mohae/joefriday) to collect the information, which was created to minimize CPU usage and memory allocations during data collection.

Autofact, on start-up, will collect information about the system on which it is running: CPU, RAM, network interfaces, Kernel, and OS. Once running, it collects, at minimum, the system's loadavg on a regular interval. It can also collect a client's CPU utilization, RAM usage, and network usage. This data is either collected locally as JSON or sent to Autofactory as Flatbuffer serialized bytes.

Communications between the server and client are via websockets and most messages are serialized using [flatbuffers.](https://google.github.io/flatbuffers/)  An inventory of clients is persisted using [boltdb.](https://github.com/boltdb/bolt).

When Autofactory is used, it can either save the data to [InfluxDB](https://influxdata.com) or as JSON to `stdout` or some other output destination.

## Optional Dependencies
[InfluxDB](https://influxdata.com/) can be used to store the facts.  Currently, it is assumed that InfluxDB is on the localhost and listening on port `8086`.

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
[Autofactory](https://github.com/mohae/autofact/tree/master/cmd/autofactory) is the optional server.  By default, it listens on `:8675` and processes incoming messages.

When a client connects, it responds with the client configuration, which defines what data is to be collected and the invterval of its colleciton. If the client is a new client, the client's ID will also be sent.

This component is not necessary if autofact is running in serverless mode.

## autofact
[Autofact](https://github.com/mohae/autofact/tree/master/cmd/autofactor) runs on client nodes. This is the only required component. It can be run in serverless mode, which will output the data, as JSON, to a local destination, which defaults to `stdout`. In client-server mode, it sends the collected data to the Autofactory server.

## Notes
This is a work in progress.

At this point in time, nothing is encrypted and `ws` is being used so definitely don't use this where communications will go over public networks.

Currently, only Linux systems are supported and this has only been tested on Debian Jessie.

Autofact may support other serialization formats with [protocol buffers](https://developers.google.com/protocol-buffers/) being the most likely.
