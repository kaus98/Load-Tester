package main

import (
	l "loadTesting/bin"
)

func main() {
	lb := l.LoadBalancer{}
	lb.PopulateConfig()
	go lb.GetStatus()
	go lb.MonitorRes()
	for i := 0; i < lb.RoutinesCount; i++ {
		lb.WG.Add(1)
		go lb.SThread()
	}

	lb.WG.Wait()
	lb.GetStatusFinished()
}
