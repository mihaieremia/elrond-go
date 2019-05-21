package ntp

import (
	"fmt"
	"sync"
	"time"

	"github.com/beevik/ntp"
	beevikntp "github.com/beevik/ntp"
)

// totalRequests defines the number of requests made to determine an accurate clock offset
const totalRequests = 10

// NTPOptions defines configuration options for an NTP query
type NTPOptions struct {
	Host         string
	Version      int
	LocalAddress string
	Timeout      time.Duration
	Port         int
}

// NewNTPOptions creates a new NTPOptions object.
func NewNTPOptions() NTPOptions {
	// TODO Read these values from configurations.
	return NTPOptions{
		Host:         "127.0.0.1",
		Port:         1123,
		Version:      0,
		LocalAddress: "",
		Timeout:      0}
}

// queryNTP wraps beevikntp.QueryWithOptions, in order to use NTPOptions, which
// contains both Host and Port, unlike beevikntp.QueryOptions.
func queryNTP(options NTPOptions) (*ntp.Response, error) {
	queryOptions := beevikntp.QueryOptions{
		Timeout:      options.Timeout,
		Version:      options.Version,
		LocalAddress: options.LocalAddress,
		Port:         options.Port}
	return beevikntp.QueryWithOptions(options.Host, queryOptions)
}

// syncTime defines an object for time synchronization
type syncTime struct {
	mut         sync.RWMutex
	clockOffset time.Duration
	syncPeriod  time.Duration
	query       func(options NTPOptions) (*ntp.Response, error)
}

// NewSyncTime creates a syncTime object
func NewSyncTime(syncPeriod time.Duration) *syncTime {
	s := syncTime{clockOffset: 0, syncPeriod: syncPeriod, query: queryNTP}
	return &s
}

// StartSync method does the time synchronization at every syncPeriod time elapsed. This should be started
// as a go routine
func (s *syncTime) StartSync() {
	for {
		s.sync()
		time.Sleep(s.syncPeriod)
	}
}

// sync method does the time synchronization and sets the current offset difference between local time
// and server time with which it has done the synchronization
func (s *syncTime) sync() {
	if s.query != nil {
		clockOffsetSum := time.Duration(0)
		succeededRequests := 0

		for i := 0; i < totalRequests; i++ {
			r, err := s.query(NewNTPOptions())

			if err != nil {
				continue
			}

			succeededRequests++
			clockOffsetSum += r.ClockOffset
		}

		if succeededRequests > 0 {
			averrageClockOffset := time.Duration(int64(clockOffsetSum) / int64(succeededRequests))
			s.setClockOffset(averrageClockOffset)
		}
	}
}

// ClockOffset method gets the current time offset
func (s *syncTime) ClockOffset() time.Duration {
	s.mut.RLock()
	clockOffset := s.clockOffset
	s.mut.RUnlock()

	return clockOffset
}

func (s *syncTime) setClockOffset(clockOffset time.Duration) {
	s.mut.Lock()
	s.clockOffset = clockOffset
	s.mut.Unlock()
}

// FormattedCurrentTime method gets the formatted current time on which is added a given offset
func (s *syncTime) FormattedCurrentTime() string {
	return s.formatTime(s.CurrentTime())
}

// formatTime method gets the formatted time from a given time
func (s *syncTime) formatTime(time time.Time) string {
	str := fmt.Sprintf("%.4d-%.2d-%.2d %.2d:%.2d:%.2d.%.9d ", time.Year(), time.Month(), time.Day(), time.Hour(),
		time.Minute(), time.Second(), time.Nanosecond())
	return str
}

// CurrentTime method gets the current time on which is added the current offset
func (s *syncTime) CurrentTime() time.Time {
	return time.Now().Add(s.clockOffset)
}
