package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/golang/glog"
	"github.com/mediocregopher/radix.v2/redis"
)

const (
	COLLECTSTATS_INVALID_INPUT        = "Invalid Endpoint"
	COLLECTSTATS_SERVER_NOT_REACHABLE = "Redis Server Not Reachable"
)

type Redis struct {
	EndPoint    string        //End point of the redis-server
	Role        string        //Role of this redis-sever Master or Slave
	LastUpdated time.Duration // When did last sync happened
	SyncBytes   int64         //How much of data did this sync
	SlaveOf     string        //Masters endpoint
	Client      *redis.Client //Redis client
}

//
func CollectStatsAll(EndPoints []string) []*Redis {

	var Servers []*Redis

	var wg sync.WaitGroup
	var lck sync.Mutex

	for _, S := range EndPoints {
		log.Infof("Processing %v", S)
		wg.Add(1)
		go func(S string) {
			defer wg.Done()
			R, err := CollectStats(S)
			if err == nil {
				lck.Lock()
				Servers = append(Servers, R)
				lck.Unlock()
			} else {
				log.Warningf("Error collecting stats for %v Error=%v", S, err)
			}
		}(S)

	}
	wg.Wait()
	return Servers
}

//CollectStats This function will take the endpoint
func CollectStats(EndPoint string) (*Redis, error) {

	var R Redis

	IpPort := strings.Split(EndPoint, ":")
	if len(IpPort) != 2 {
		return nil, fmt.Errorf(COLLECTSTATS_INVALID_INPUT)
	}

	//Try to connect to the redis-servers
	C, err := redis.Dial("tcp", EndPoint)
	if err != nil {
		log.Infof("CollectStats() %s Error:%v", COLLECTSTATS_SERVER_NOT_REACHABLE, err)
		return nil, fmt.Errorf(COLLECTSTATS_SERVER_NOT_REACHABLE)
	}
	Res := C.Cmd("INFO", "REPLICATION")

	log.Infof("CollectStats(%s)=%v", EndPoint, Res.String())

	R.Client = C
	return &R, nil
}

func main() {

	//Parse command line arguments
	ServersEndPoint := os.Args[1:]
	log.Infof("Supplied Arguments are %v", ServersEndPoint)

	//Collect stats on all the redis-servers supplied
	Servers := CollectStatsAll(ServersEndPoint)

	//collect Redis endpoints
	log.Infof("Available Servers are %v", Servers)

	//Does it really need a master

	log.Infof("Redis-Sentinal-micro Finished")

}
