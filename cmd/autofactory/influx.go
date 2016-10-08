package main

import (
	"fmt"

	client "github.com/influxdata/influxdb/client/v2"
	"github.com/uber-go/zap"
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
				log.Fatal(
					"series data channel is closed",
					zap.String("db", "influxdb"),
					zap.String("dbname", c.DBName),
				)
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
			for _, v := range series.Data {
				bp.AddPoint(v)
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
