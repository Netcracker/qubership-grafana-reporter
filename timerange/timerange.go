// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package timerange

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type TimerangeData struct {
	From     string
	To       string
	DateFrom time.Time
	DateTo   time.Time
}

var Format = "2006-01-02 15:04:05 +0000 UTC"
var intervalsInSeconds = map[string]int{
	"y": 31536000,
	"M": 2592000,
	"w": 604800,
	"d": 86400,
	"h": 3600,
	"m": 60,
	"s": 1}

var regExpressionRelative = regexp.MustCompile(`now-(?P<Count>\d*)(?P<Unit>\w)/?(?P<Boundary>\w?)`)
var regExpressionBoundary = regexp.MustCompile(`now/(?P<Unit>\w)`)

func RelativeTimeToTimestamp(currentTime time.Time, relativeTime string, fromOrTo string) (time.Time, error) {
	if ts := isTimestamp(relativeTime); ts != nil {
		tsTime, ok := ts.(time.Time)
		if !ok {
			return time.Time{}, fmt.Errorf("could not cast timestamp to time.Time")
		}
		return tsTime, nil
	}
	if isRelativeTime(relativeTime) {
		switch {
		case strings.HasPrefix(relativeTime, "now/"):
			return parseBoundaryTime(currentTime, relativeTime, fromOrTo)
		case strings.HasPrefix(relativeTime, "now-"):
			return parseRelativeTime(currentTime, relativeTime, fromOrTo)
		case strings.EqualFold(relativeTime, "now"):
			return currentTime, nil
		default:
			return time.Time{}, fmt.Errorf("could not cast relative time to time.Time")
		}
	} else {
		return time.Time{}, fmt.Errorf("could not cast relative time to time.Time")
	}
}

func isTimestamp(relativeTime string) interface{} {
	tstemp, err := strconv.ParseInt(relativeTime, 10, 64)
	if err != nil {
		return nil
	}
	tm := time.Unix(tstemp, 0)
	return tm
}

func isRelativeTime(time string) bool {
	return strings.Contains(time, "now")
}

func parseBoundaryTime(currentTime time.Time, relativeTime string, fromOrTo string) (time.Time, error) {
	match := regExpressionBoundary.FindStringSubmatch(relativeTime)
	var instantTime time.Time
	if match[1] != "" {
		year, month, day := currentTime.Date()
		hour := currentTime.Hour()
		minutes := currentTime.Minute()
		seconds := currentTime.Second()
		var weekday time.Duration
		if strings.EqualFold(fromOrTo, "from") {
			switch match[1] {
			case "m":
				seconds = 0
			case "h":
				seconds = 0
				minutes = 0
			case "d":
				seconds = 0
				minutes = 0
				hour = 0
			case "w":
				seconds = 0
				minutes = 0
				hour = 0
				weekday = time.Duration(currentTime.Weekday())
			case "M":
				seconds = 0
				minutes = 0
				hour = 0
				day = 1
			case "y":
				seconds = 0
				minutes = 0
				hour = 0
				day = 1
				month = 1
			}
			instantTime = time.Date(year, month, day, hour, minutes, seconds, 0, time.UTC)
			if match[1] == "w" {
				instantTime = instantTime.Add(-1 * weekday * 24 * time.Hour)
			}
			return instantTime, nil
		} else {
			switch match[1] {
			case "h":
				seconds = 59
				minutes = 59
			case "d":
				seconds = 59
				minutes = 59
				hour = 23
			case "w":
				seconds = 59
				minutes = 59
				hour = 23
				weekday = time.Duration(currentTime.Weekday())
			case "M":
				seconds = 59
				minutes = 59
				hour = 23
				firstDayOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
				lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)
				day = lastDayOfMonth.Day()
			case "y":
				seconds = 59
				minutes = 59
				hour = 23
				firstDayOfMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
				lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)
				day = lastDayOfMonth.Day()
				month = 12
			}
			instantTime = time.Date(year, month, day, hour, minutes, seconds, 1e9-1, time.UTC)
			if match[1] == "w" {
				instantTime = instantTime.Add((6 - weekday) * 24 * time.Hour)
			}
			return instantTime, nil
		}
	} else {
		return time.Time{}, fmt.Errorf("time value is not valid: %s", relativeTime)
	}
}
func parseRelativeTime(currentTime time.Time, relativeTime string, fromOrTo string) (time.Time, error) {
	var tsTime time.Time
	match := regExpressionRelative.FindStringSubmatch(relativeTime)
	var count, secondsUnit int
	if match[1] != "" {
		var err error
		count, err = strconv.Atoi(match[1])
		if err != nil {
			return time.Time{}, err
		}
	} else {
		count = 1
	}
	if match[3] != "" {
		if strings.EqualFold(fromOrTo, "from") {
			switch match[3] {
			case "d":
				tsTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -count)
			case "w":
				weekday := time.Duration(currentTime.Weekday())
				tsTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC).Add(-1*weekday*24*time.Hour).AddDate(0, 0, -7*count)
			case "M":
				tsTime = time.Date(currentTime.Year(), currentTime.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -count, 0)
			case "y":
				tsTime = time.Date(currentTime.Year(), 1, 1, 0, 0, 0, 0, time.UTC).AddDate(-count, 0, 0)
			default:
				return time.Time{}, fmt.Errorf("time value is not valid: %s", relativeTime)
			}
			return tsTime, nil
		} else {
			switch match[3] {
			case "d":
				tsTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 1e9-1, time.UTC).AddDate(0, 0, -count)
			case "w":
				weekday := time.Duration(currentTime.Weekday())
				tsTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 1e9-1, time.UTC).Add((6-weekday)*24*time.Hour).AddDate(0, 0, -7*count)
			case "M":
				tsTime = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 1e9-1, time.UTC).AddDate(0, -count+1, -currentTime.Day())
			case "y":
				tsTime = time.Date(currentTime.Year(), 12, currentTime.Day(), 23, 59, 59, 1e9-1, time.UTC).AddDate(-count, 1, -currentTime.Day())
			default:
				return time.Time{}, fmt.Errorf("time value is not valid: %s", relativeTime)
			}
			return tsTime, nil
		}
	}
	if match[2] != "" {
		switch match[2] {
		case "y":
			tsTime = currentTime.AddDate(-count, 0, 0)
		case "M":
			tsTime = currentTime.AddDate(0, -count, 0)
		default:
			secondsUnit = intervalsInSeconds[match[2]]
			if secondsUnit == 0 {
				return time.Time{}, fmt.Errorf("time value is not valid: %s", relativeTime)
			}
			tsTime = currentTime.Add(time.Duration(-count*secondsUnit) * time.Second)
		}
		return tsTime, nil
	} else {
		return time.Time{}, fmt.Errorf("time value is not valid: %s", relativeTime)
	}
}
