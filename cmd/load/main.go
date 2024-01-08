package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type timeReport struct {
	start time.Time
	end   time.Time
	dur   time.Duration
}

type TimeManager struct {
	start time.Time
	times []timeReport
}

func NewTimeManager() *TimeManager {
	return &TimeManager{
		start: time.Now(),
	}
}

func (t *TimeManager) AddTime(tr timeReport) {
	t.times = append(t.times, tr)
}

func (t *TimeManager) Stat() {
	count := len(t.times)

	rps := float64(count) / time.Since(t.start).Seconds()

	avg := time.Duration(0)
	for _, tr := range t.times {
		avg += tr.dur
	}
	avg = avg / time.Duration(count)

	min, max := time.Duration(1000000000000000000), time.Duration(0)
	for _, tr := range t.times {
		if tr.dur < min {
			min = tr.dur
		}
		if tr.dur > max {
			max = tr.dur
		}
	}

	fmt.Printf("count: %d\trps: %f\tavg: %s\tmin: %s\tmax: %s\n", count, rps, avg, min, max)
}

func (t *TimeManager) AutoStat() {
	go func() {
		for {
			time.Sleep(time.Second)
			t.Stat()
		}
	}()
}

func main() {
	tm := NewTimeManager()
	tm.AutoStat()
	limit := make(chan int, 100)
	for {
		limit <- 1
		go func() {
			st := time.Now()
			tr := timeReport{
				start: st,
			}
			makeRequest()
			tr.end = time.Now()
			tr.dur = tr.end.Sub(tr.start)
			tm.AddTime(tr)
			<-limit
		}()
	}
}

func makeRequest() {
	req, err := http.NewRequest("GET", "http://localhost:5000/live/09248ef6-c401-4601-8928-5964d61f2c61", nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("%d error: %s\n", resp.StatusCode, string(b))
	}

}
