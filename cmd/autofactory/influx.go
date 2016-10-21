package main

import (
	"fmt"
	"os"

	client "github.com/influxdata/influxdb/client/v2"
	"github.com/uber-go/zap"
)

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
		pointsCh: make(chan []*influx.Point, 100),
		doneCh:   make(chan struct{}),
	}, nil
}

// InfluxClient manages the connection and interactions with the target
// InfluxDB.
type InfluxClient struct {
	DBName    string
	Conn      client.Client
	Precision string
	pointsCh  chan []*influx.Point
	doneCh    chan struct{}
}

// Write writes received BatchPoints to the database
func (c *InfluxClient) Write() {
	defer c.Conn.Close()
	defer close(c.doneCh)
	for {
		select {
		case points, ok := <-c.pointsCh:
			if !ok {
				log.Error(
					"influx points channel closed",
					zap.String("db", "influxdb"),
					zap.String("dbname", c.DBName),
				)
				srvr.Bolt.Close() // Close Bolt because defers don't run on os.Exit
				CloseLog()        // Close log because defers don't run on os.Exit
				os.Exit(1)
			}
			// TODO: work out error handling
			if series.err != nil {
				log.Error(
					series.err.Error(),
					zap.String("op", "process series data"),
					zap.String("db", "influxdb"),
				)
				continue
			}
			// create the batchpoint from the data
			bp, err := client.NewBatchPoints(client.BatchPointsConfig{
				Database:  c.DBName,
				Precision: c.Precision,
			})
			if err != nil {
				log.Error(
					err.Error(),
					zap.String("op", "create batch points"),
					zap.String("db", "influxdb"),
				)
				continue
			}
			for _, p := range points {
				bp.AddPoint(p)
			}
			err = c.Conn.Write(bp)
			if err != nil {
				log.Error(
					err.Error(),
					zap.String("op", "write series data"),
					zap.String("db", "influxdb"),
				)
				continue
			}
		case _, ok := <-c.doneCh:
			if !ok {
				return
			}
		}
	}
}
