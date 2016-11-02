autofactory
===========

Autofactory is the [autofact](https://github.com/mohae/autofact) server component.  `Autofact` clients connect to the `autofactory` and send it their collected data.  

At minimum, `healthbeat` information is collected from the client. The `healthbeat` is a pull operation and is how Autofactory checks to see if a client is still connected, or if it has gone away, for whatever reason.

All other data collected from the client, other than the client's system information, which is collected during the client connection process, is pushed to Autofactory by the client.

Autofactory sends newly connected clients their configuration.

## Data output
The collected data can either be written to a file, as JSON, or stored in [InfluxDB](https://influxdata.com). The `datadestination` flag specifies the output for the data, `file` is the default. For InfluxDB use `influxdb`.

When the output is `file`, the default is `stdout`, for a specific location use the `dataout` flag.

## Logging
Log entries are written as JSON with `stderr` as the default destination. The log destination can be set using `logout`.

## TODO

* Add collection of disk information  
* Improve Client attributes: e.g. Group, Role, Datacenter, or should it be a map of attributes?  
* Make data collection intervals configurable:  
  * Per client  
  * Per attribute: e.g. Group, role, etc.  
* ...
