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
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

// Test format now/[dwMy]
func Test_TimerangeToTimestamp(t *testing.T) {
	for _, testUnit := range testData {
		t.Run(testUnit.TestName, func(t *testing.T) {
			fromTS, err := RelativeTimeToTimestamp(now, testUnit.FromTR, "from")
			assert.NoError(t, err)
			assert.Equal(t, testUnit.FromTS, fromTS.Format(Format))
			toTS, err := RelativeTimeToTimestamp(now, testUnit.ToTR, "to")
			assert.NoError(t, err)
			assert.Equal(t, testUnit.ToTS, toTS.Format(Format))
		})
	}
}

type TimerangeTestData struct {
	TestName string
	FromTR   string
	ToTR     string
	FromTS   string //datetime
	ToTS     string
}

var now = time.Date(2024, 1, 25, 14, 43, 12, 0, time.UTC)
var nowString = now.Format(Format) //"2024-01-25 14:43:12 +0000 UTC"
var testData = []TimerangeTestData{
	//format now/[dwMy]
	{"From start of the day to now", "now/d", "now", "2024-01-25 00:00:00 +0000 UTC", nowString},
	{"From start of the day to the end of the day (in future)", "now/d", "now/d", "2024-01-25 00:00:00 +0000 UTC", "2024-01-25 23:59:59 +0000 UTC"},
	{"From the start of the current week to now", "now/w", "now", "2024-01-21 00:00:00 +0000 UTC", nowString},
	{"From the start of the current week to the end of the current week (in future)", "now/w", "now/w", "2024-01-21 00:00:00 +0000 UTC", "2024-01-27 23:59:59 +0000 UTC"},
	{"From the start of the month to now", "now/M", "now", "2024-01-01 00:00:00 +0000 UTC", nowString},
	{"From the start of the month to the end of the month (in future)", "now/M", "now/M", "2024-01-01 00:00:00 +0000 UTC", "2024-01-31 23:59:59 +0000 UTC"},
	{"From the start of the year to now", "now/y", "now", "2024-01-01 00:00:00 +0000 UTC", nowString},
	{"From the start of the year to the end of the year (in future)", "now/y", "now/y", "2024-01-01 00:00:00 +0000 UTC", "2024-12-31 23:59:59 +0000 UTC"},
	//format now-\d*[mhdwMy]
	{"Last 5 minutes", "now-5m", "now", "2024-01-25 14:38:12 +0000 UTC", nowString},
	{"Last 15 minutes", "now-15m", "now", "2024-01-25 14:28:12 +0000 UTC", nowString},
	{"Last 30 minutes", "now-30m", "now", "2024-01-25 14:13:12 +0000 UTC", nowString},
	{"Last hour", "now-1h", "now", "2024-01-25 13:43:12 +0000 UTC", nowString},
	{"Last 3 hours", "now-3h", "now", "2024-01-25 11:43:12 +0000 UTC", nowString},
	{"Last 6 hours", "now-6h", "now", "2024-01-25 08:43:12 +0000 UTC", nowString},
	{"Last 12 hours", "now-12h", "now", "2024-01-25 02:43:12 +0000 UTC", nowString},
	{"Last 24 hours", "now-24h", "now", "2024-01-24 14:43:12 +0000 UTC", nowString},
	{"Last 24 hours", "now-d", "now", "2024-01-24 14:43:12 +0000 UTC", nowString},
	{"Last 24 hours", "now-1d", "now", "2024-01-24 14:43:12 +0000 UTC", nowString},
	{"Last 2 days", "now-2d", "now", "2024-01-23 14:43:12 +0000 UTC", nowString},
	{"Last 7 days", "now-7d", "now", "2024-01-18 14:43:12 +0000 UTC", nowString},
	{"Last 30 days", "now-30d", "now", "2023-12-26 14:43:12 +0000 UTC", nowString},
	{"Last 90 days", "now-90d", "now", "2023-10-27 14:43:12 +0000 UTC", nowString},
	{"Last week", "now-w", "now", "2024-01-18 14:43:12 +0000 UTC", nowString},
	{"Last 2 weeks", "now-2w", "now", "2024-01-11 14:43:12 +0000 UTC", nowString},
	{"Last 6 months", "now-6M", "now", "2023-07-25 14:43:12 +0000 UTC", nowString},
	{"Last year", "now-1y", "now", "2023-01-25 14:43:12 +0000 UTC", nowString},
	{"Last 2 years", "now-2y", "now", "2022-01-25 14:43:12 +0000 UTC", nowString},
	//format now-\d*[dwMy]/[dwMy]
	{"Yesterday", "now-1d/d", "now-1d/d", "2024-01-24 00:00:00 +0000 UTC", "2024-01-24 23:59:59 +0000 UTC"},
	{"Day before yesterday", "now-2d/d", "now-2d/d", "2024-01-23 00:00:00 +0000 UTC", "2024-01-23 23:59:59 +0000 UTC"},
	{"This day last week", "now-7d/d", "now-7d/d", "2024-01-18 00:00:00 +0000 UTC", "2024-01-18 23:59:59 +0000 UTC"},
	{"Previous week", "now-1w/w", "now-1w/w", "2024-01-14 00:00:00 +0000 UTC", "2024-01-20 23:59:59 +0000 UTC"},
	{"The 2nd week ago", "now-2w/w", "now-2w/w", "2024-01-07 00:00:00 +0000 UTC", "2024-01-13 23:59:59 +0000 UTC"},
	{"Previous month", "now-1M/M", "now-1M/M", "2023-12-01 00:00:00 +0000 UTC", "2023-12-31 23:59:59 +0000 UTC"},
	{"The 2nd month ago", "now-2M/M", "now-2M/M", "2023-11-01 00:00:00 +0000 UTC", "2023-11-30 23:59:59 +0000 UTC"},
	{"Previous year", "now-1y/y", "now-1y/y", "2023-01-01 00:00:00 +0000 UTC", "2023-12-31 23:59:59 +0000 UTC"},
	{"The 2nd year ago", "now-2y/y", "now-2y/y", "2022-01-01 00:00:00 +0000 UTC", "2022-12-31 23:59:59 +0000 UTC"},
}
