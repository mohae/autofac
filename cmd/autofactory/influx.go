package main

import (
	"fmt"
	"os"

	client "github.com/influxdata/influxdb/client/v2"
)

// series holds a series of data points for Influx.
type Series struct {
	Data []*client.Point
	err  error
}

// newInfluxClient connects to the database with the passed info andd returns
// the InfluxClient.  If an error occurs, that will be returned.
func newInfluxClient(name, addr, user, password string) (*InfluxClient, error) {
	cl, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     addr,
		Username: user,
		Password: password,
	})
	if err != nil {
		return nil, fmt.Errorf("InfluxDB: client connect failed: %s", err)
	}
	return &InfluxClient{
		DBName:   name,
		Conn:     cl,
		seriesCh: make(chan Series, 100),
		doneCh:   make(chan struct{}),
	}, nil
}

// InfluxClient manages the connection and interactions with the target
// InfluxDB.
type InfluxClient struct {
	DBName    string
	Conn      client.Client
	Precision string
	seriesCh  chan Series
	doneCh    chan struct{}
}

// Write writes received BatchPoints to the database
func (c *InfluxClient) Write() {
	defer c.Conn.Close()
	defer close(c.doneCh)
	for {
		select {
		case series, ok := <-c.seriesCh:
			if !ok {
				fmt.Fprintf(os.Stderr, "InfluxDB: series chan for %s database closed: exiting\n", c.DBName)
				return
			}
			// TODO: work out error handling
			if series.err != nil {
				fmt.Fprintf(os.Stderr, "InfluxDB: series had an error: %s", series.err)
				continue
			}
			// create the batchpoint from the data
			bp, err := client.NewBatchPoints(client.BatchPointsConfig{
				Database:  c.DBName,
				Precision: c.Precision,
			})
			if err != nil {
				fmt.Fprintf(os.Stderr, "InfluxDB: unable to create batch points for series: %s", err)
				continue
			}
			for _, v := range series.Data {
				bp.AddPoint(v)
			}
			err = c.Conn.Write(bp)
			if err != nil {
				fmt.Fprintf(os.Stderr, "InfluxDB: write of series data failed: %s", err)
				continue
			}
		case _, ok := <-c.doneCh:
			if !ok {
				return
			}
		}
	}
}
