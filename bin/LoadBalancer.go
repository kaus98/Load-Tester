package bin

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type LoadBalancer struct {
	httpAddress   string
	httpData      string
	httpMethod    string
	tlsSkipVerify bool

	CounterLimit  int
	RoutinesCount int

	Counter             int
	IntervalCounter     int
	TotalRequestTime    time.Duration
	IntervalRequestTime time.Duration
	LogTime             time.Duration
	BegTime             time.Time
	CurTime             time.Time

	httpHeader map[string]string
	Status     map[int]int

	StopCh       bool
	Response     chan ResponseDetails
	ResponseCode chan int

	WG      sync.WaitGroup
	MuxLock sync.RWMutex
}

func (lb *LoadBalancer) PopulateConfig() {
	//Endpoint details here
	lb.httpAddress = ""
	lb.httpData = ``
	lb.httpMethod = "POST"
	lb.httpHeader = make(map[string]string)
	lb.httpHeader["Content-Type"] = "text/plain"
	lb.tlsSkipVerify = true

	//Rquests Meta data here
	lb.CounterLimit = 2000 //Number of requests to send
	lb.RoutinesCount = 400 //Number of Go Routines to start
	lb.LogTime = 5         //Frequency of requests

	//Other internal data
	lb.Response = make(chan ResponseDetails)
	lb.ResponseCode = make(chan int) // Get Response code in Channel
	lb.Counter = 0                   //Handle number of completed Request
	lb.IntervalCounter = 0           //Handle number of completed Request
	lb.TotalRequestTime = 0
	lb.IntervalRequestTime = 0
	lb.StopCh = true
	lb.Status = make(map[int]int)
	lb.MuxLock = sync.RWMutex{}
	lb.BegTime = time.Now()
	lb.CurTime = time.Now()

}

func (lb *LoadBalancer) CreateClient() (*http.Client, error) {

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: lb.tlsSkipVerify},
	}

	client := &http.Client{Transport: tr}

	return client, nil
}

func (lb *LoadBalancer) CreateRequest() (*http.Request, error) {
	req, err := http.NewRequest(lb.httpMethod, lb.httpAddress, strings.NewReader(lb.httpData))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	for key, value := range lb.httpHeader {
		req.Header.Add(key, value)
	}

	return req, nil
}

func (lb *LoadBalancer) SThread() {
	defer lb.WG.Done()

	client, _ := lb.CreateClient()
	req, err := lb.CreateRequest()

	var ttime time.Time
	var rrd ResponseDetails
	sch := true

	if err != nil {
		fmt.Println(err)
		return
	}
	for {

		lb.MuxLock.RLock()
		sch = lb.StopCh
		lb.MuxLock.RUnlock()

		if !sch {
			return
		}
		ttime = time.Now()
		res, err := client.Do(req)

		if err != nil {
			fmt.Println(err)
			// return
		} else {
			rrd = ResponseDetails{ttime, time.Now(), res.StatusCode}
			lb.Response <- rrd

		}

	}
}

func (lb *LoadBalancer) MonitorRes() {

	for {
		res := <-lb.Response

		//Mapping the Request to Status Code
		if value, ok := lb.Status[res.Code]; ok {
			lb.Status[res.Code] = value + 1
		} else {
			lb.Status[res.Code] = 1
		}

		//Handling the time of request
		lb.MuxLock.Lock()

		lb.TotalRequestTime += res.EndTime.Sub(res.StartTime)
		lb.IntervalRequestTime += res.EndTime.Sub(res.StartTime)

		lb.Counter += 1
		lb.IntervalCounter += 1

		lb.MuxLock.Unlock()

		if lb.Counter >= (lb.CounterLimit - lb.RoutinesCount) {
			lb.MuxLock.Lock()
			lb.StopCh = false
			lb.MuxLock.Unlock()

		}
	}
}

func (lb *LoadBalancer) GetStatus() {
	//Properly print the details for Running jobs
	for {
		time.Sleep(lb.LogTime * time.Second)
		log.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
		//Status Code Result
		log.Printf("Current Status Codes  --> ")
		for ky, vlu := range lb.Status {
			log.Printf("\t%v : %v", ky, vlu)
		}

		lb.MuxLock.Lock()
		//Total Requests Send
		log.Printf("Total Requests Send : %v", lb.Counter)
		log.Printf("Current Requests Send : %v", lb.IntervalCounter)

		//Total Response Time Result
		log.Printf("Total Response Time per Request : %.3f sec", lb.TotalRequestTime.Seconds()/float64(lb.Counter))
		//Current Response Time Result
		log.Printf("Current Response Time per Request : %.3f sec", lb.IntervalRequestTime.Seconds()/float64(lb.IntervalCounter))

		//Current Request Rate
		log.Printf("Total Request Rate : %.2f req/sec", float64(lb.Counter/int(time.Since(lb.BegTime).Seconds())))
		log.Printf("Current Request Rate : %.2f req/sec", float64(lb.IntervalCounter/int(time.Since(lb.CurTime).Seconds())))

		lb.IntervalRequestTime = 0
		lb.IntervalCounter = 0
		lb.CurTime = time.Now()

		lb.MuxLock.Unlock()

		log.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")
	}
}

func (lb *LoadBalancer) GetStatusFinished() {

	log.Println(">>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>")
	//Status Code Result
	log.Printf("Current Status Codes  --> ")
	for ky, vlu := range lb.Status {
		log.Printf("\t%v : %v", ky, vlu)
	}

	lb.MuxLock.Lock()
	//Total Requests Send
	log.Printf("Total Requests Send : %v", lb.Counter)
	log.Printf("Total Request Rate : %.2f req/sec", float64(lb.Counter/int(time.Since(lb.BegTime).Seconds())))
	log.Printf("Total Response Time per Request : %f sec", lb.TotalRequestTime.Seconds()/float64(lb.Counter))
	lb.MuxLock.Unlock()

	log.Println("<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<<")

}
