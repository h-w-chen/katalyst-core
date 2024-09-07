/*
Copyright 2022 The Katalyst Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package reader

import (
	"context"
	"fmt"

	"github.com/bougou/go-ipmi"
	"github.com/pkg/errors"
)

type ipmiPowerReader struct {
	ipmiClient  *ipmi.Client
	powerSensor *Sensor
}

func getPowerSDR(client *ipmi.Client) (*SDR, error) {
	// most machines have Total_Power as the sensor name, some have Sys_Total_Power
	if sdr, err := client.GetSDRBySensorName("Total_Power"); err == nil {
		return sdr, nil
	}
	return client.GetSDRBySensorName("Sys_Total_Power")
}

func getPowerSensor(client *ipmi.Client) (*Sensor, error) {
	// sometimes it fails to get sdr; retry a couple of times before giving up
	var sdr *SDR
	var err error

	for retry := 0; retry < 3; retry += 1 {
		sdr, err = getPowerSDR(client)
		if err == nil {
			break
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "ipmi searching for total power sensor failed")
	}

	var sensor *ipmi.Sensor
	sensor, err = sdrToSensor(sdr)
	if err != nil {
		return nil, errors.Wrap(err, "ipmi locating power sensor failed")
	}

	return sensor, nil
}

func (pr *ipmiPowerReader) Init() error {
	client, err := ipmi.NewOpenClient()
	if err != nil {
		return errors.Wrap(err, "ipmi creating client failed")
	}

	if err := client.Connect(); err != nil {
		return errors.Wrap(err, "ipmi connecting client failed")
	}

	sensor, err := getPowerSensor(client)
	if err != nil {
		return errors.Wrap(err, "ipmi locating power sensor failed")
	}

	pr.powerSensor = sensor
	pr.ipmiClient = client
	return nil
}

func (pr *ipmiPowerReader) Cleanup() {
	if pr.ipmiClient != nil {
		_ = pr.ipmiClient.Close()
		pr.ipmiClient = nil
	}
}

func (pr *ipmiPowerReader) Get(_ context.Context) (int, error) {
	resp, err := pr.ipmiClient.GetSensorReading(pr.powerSensor.Number)
	if err != nil {
		return 0, errors.Wrap(err, "ipmi reading power sensor failed")
	}

	return int(pr.powerSensor.ConvertReading(resp.Reading)), nil
}

type (
	SDR    = ipmi.SDR
	Sensor = ipmi.Sensor
)

const (
	SDRRecordTypeFullSensor    = ipmi.SDRRecordTypeFullSensor
	SDRRecordTypeCompactSensor = ipmi.SDRRecordTypeCompactSensor
)

// sdrToSensor is almost authentic copy from github.com/bougou/go-ipmi,
// function (c *ipmi.Client) sdrToSensor, with extras truncated;
// it is not exposed there, and needed here for sensor conversion
func sdrToSensor(sdr *SDR) (*Sensor, error) {
	if sdr == nil {
		return nil, fmt.Errorf("nil sdr parameter")
	}

	sensor := &Sensor{
		SDRRecordType:    sdr.RecordHeader.RecordType,
		HasAnalogReading: sdr.HasAnalogReading(),
	}

	switch sdr.RecordHeader.RecordType {
	case SDRRecordTypeFullSensor:
		sensor.Number = uint8(sdr.Full.SensorNumber)
		sensor.Name = string(sdr.Full.IDStringBytes)
		sensor.SensorUnit = sdr.Full.SensorUnit
		sensor.SensorType = sdr.Full.SensorType
		sensor.EventReadingType = sdr.Full.SensorEventReadingType
		sensor.SensorInitialization = sdr.Full.SensorInitialization
		sensor.SensorCapabilities = sdr.Full.SensorCapabilities

		sensor.Threshold.LinearizationFunc = sdr.Full.LinearizationFunc
		sensor.Threshold.ReadingFactors = sdr.Full.ReadingFactors

	case SDRRecordTypeCompactSensor:
		sensor.Number = uint8(sdr.Compact.SensorNumber)
		sensor.Name = string(sdr.Compact.IDStringBytes)
		sensor.SensorUnit = sdr.Compact.SensorUnit
		sensor.SensorType = sdr.Compact.SensorType
		sensor.EventReadingType = sdr.Compact.SensorEventReadingType
		sensor.SensorInitialization = sdr.Compact.SensorInitialization
		sensor.SensorCapabilities = sdr.Compact.SensorCapabilities

	default:
		return nil, fmt.Errorf("only support Full or Compact SDR record type, input is %s", sdr.RecordHeader.RecordType)
	}

	return sensor, nil
}

func NewIPMIPowerReader() PowerReader {
	return &ipmiPowerReader{}
}
