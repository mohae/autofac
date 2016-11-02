autofact
===========

Autofact collects basic data about the system on which it runs for system monitoring purposes. Autofact can be be run serverless or connected to [Autofactory](https://github.com/mohae/autofact/tree/master/cmd/autofactory).

All Autofact related configurations and files are written to `$HOME/.autofact` This can be overriden by setting the `AUTOFACT_PATH` env variable.

Currently, this only runs on amd64 linux systems.

Collection of disk information has not been implemented.

## Data collected
On start-up, Autofact collects information about the system on which it is running. This information includes CPUs, Memory, Network Interfaces, Kernel, and OS.

Autofact has a healthbeat, which is the current `loadavg` information along with the ability to collect information about memory usage, cpu usage, and network usage. Each of these datasets can be collected on their own interval. Only the healthbeat is always collected.

If Autofact is being run in serverless mode, the collection periods are specified in `autoocollect.json`. If one doesn't exist, Autofact will generate one with its defaults. If Autofact is connecting to a server, it will always get it's configuration from the server; local settings will be ignored.

## Operation modes
### Serverless
Autofact can be run in serverless mode by passing the `serverless` flag.  The collected datapoints will be written to a local resource as JSON. By default, collected data is written to `stdout` and any errors are written to `stderr`. Both the data destination and log destination can be set by passing the `dataout` and `logout` flags, respectively.

### Client - Server
When Autofact is running as a client connected to a server, Autofactory, it will connect to the Autofactory instance. If this is the first time it has connected, Autofactory will give it its ClientID, otherwise, it sends Autofactory its ClientID.

After a successful connection, it will send the Autofactory its system information as JSON. Other than system information, all other collected data is sent to the server as Flatbuffer serialized bytes.

In the future, other serialization formats may be supported.

#### Healthbeat
Autofactory, on a given interval, will request a healthbeat from the Autofact client. The healthbeat data is the client's current `loadavg` data. This is a pull operation because that is how Autofactory checks to see if a client is still running or if it has gone away.

#### CPUUtilization, Meminfo, NetUsage
The other datapoints that are to be collected are pushed to the Autofactory server on the configured interval for that datapoint.
