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

package spec

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
)

type (
	PowerAlert string
	InternalOp int
)

const (
	// authentic power alert code
	PowerAlertS0 PowerAlert = "s0"
	PowerAlertP0 PowerAlert = "p0"
	PowerAlertP1 PowerAlert = "p1"
	PowerAlertP2 PowerAlert = "p2"

	// PowerAlertOK is derivative power alert code which corresponds to NON-existent power alert annotation
	PowerAlertOK PowerAlert = "ok"

	InternalOpAuto     InternalOp = 0 // default op allowing plugin makes decision by itself
	InternalOpThrottle InternalOp = 1 // op demanding plugin choose throttling compute resources only
	InternalOpEvict    InternalOp = 2 // op demanding plugin choose eviction only
	InternalOpFreqCap  InternalOp = 4 // op demanding plugin to choose cpu frequency capping only
	InternalOpNoop     InternalOp = 8 // op demanding plugin not making any policy

	AnnoKeyPowerAlert      = "tce.kubernetes.io/power-alert"
	AnnoKeyPowerBudget     = "tce.kubernetes.io/power-budget"
	AnnoKeyPowerAlertTime  = "tce.kubernetes.io/power-alert-time"
	AnnoKeyPowerInternalOp = "tce.kubernetes.io/power-internal-op"
)

var (
	powerAlertResponseTime = map[PowerAlert]time.Duration{
		PowerAlertS0: time.Minute * 2,
		PowerAlertP0: time.Minute * 30,
		PowerAlertP1: time.Hour * 1,
		PowerAlertP2: time.Hour * 4,
	}

	unknownAlertError = errors.New("unknown alert")
)

func GetPowerAlertResponseTimeLimit(alert PowerAlert) (time.Duration, error) {
	alert = PowerAlert(strings.ToLower(string(alert)))
	resp, ok := powerAlertResponseTime[alert]
	if !ok {
		return time.Duration(0), unknownAlertError
	}
	return resp, nil
}

func (o InternalOp) String() string {
	switch o {
	case InternalOpThrottle:
		return "Throttle"
	case InternalOpEvict:
		return "Evict"
	case InternalOpFreqCap:
		return "FreqCap"
	case InternalOpNoop:
		return "Noop"
	default:
		return fmt.Sprintf("%d", int(o))
	}
}

type PowerSpec struct {
	Alert      PowerAlert
	Budget     int
	InternalOp InternalOp
	AlertTime  time.Time
}

func GetPowerSpec(node *v1.Node) (*PowerSpec, error) {
	alert := PowerAlert(node.Annotations[AnnoKeyPowerAlert])
	if len(alert) == 0 {
		return &PowerSpec{
			Alert:      PowerAlertOK,
			Budget:     0,
			InternalOp: 0,
		}, nil
	}

	// uniformly convert alert level input to lower case just in case
	alert = PowerAlert(strings.ToLower(string(alert)))

	// input float number like 611.31 is allowed
	budget, err := strconv.ParseFloat(node.Annotations[AnnoKeyPowerBudget], 32)
	if err != nil {
		return nil, errors.Wrap(err, "budget is not a numeral")
	}

	internalOp := InternalOpAuto
	if len(node.Annotations[AnnoKeyPowerInternalOp]) > 0 {
		code, err := strconv.Atoi(node.Annotations[AnnoKeyPowerInternalOp])
		if err != nil {
			return nil, errors.Wrap(err, "op is not a digit")
		}
		internalOp = InternalOp(code)
	}

	alertTimeStr := node.Annotations[AnnoKeyPowerAlertTime]
	alertTime, err := time.Parse(time.RFC3339, alertTimeStr)
	if err != nil {
		return nil, errors.Wrap(err, "alert time is not in RFC3339 format")
	}
	return &PowerSpec{
		Alert:      alert,
		Budget:     int(budget),
		InternalOp: internalOp,
		AlertTime:  alertTime,
	}, nil
}
