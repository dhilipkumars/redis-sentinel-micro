package main

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"sort"

	log "github.com/golang/glog"
	"github.com/mediocregopher/radix.v2/redis"
	"strconv"
)

const (
	COLLECTSTATS_INVALID_INPUT        = "Invalid Endpoint"
	COLLECTSTATS_SERVER_NOT_REACHABLE = "Redis Server Not Reachable"
)

type Redis struct {
	EndPoint         string        //End point of the redis-server
	Role             string        //Role of this redis-sever Master or Slave
	LastUpdated      int           // When did last sync happened seconds
	MasterDownSince  int		//Since how long the master is not reachable?
	SyncBytes        int64         //How much of data did this sync
	MasterHost          string        //Masters ip addres
	MasterPort	  string	 //Master port
	Priority         int           //Slave priority
	MasterLinkStatus bool          //true means up false mean down
	Client           *redis.Client //Redis client
}

type RedisSlaves []*Redis

func (rs RedisSlaves) Len() int {return len(rs)}
func (rs RedisSlaves) Swap (i int, j int) {
	var tmp *Redis
	tmp = rs[i]
	rs[i] = rs[j]
	rs[j] = tmp
}

func (rs RedisSlaves) Less (i int, j int) bool {

	if rs[i].Priority < rs[j].Priority {
		return true
	}

	if rs[i].SyncBytes < rs[j].SyncBytes {
		return true
	}

	if rs[i].LastUpdated > rs[j].LastUpdated {
		return true
	}

	return false
}

func (R *Redis) ParseResponse(Res string) bool {

	res := strings.Split(Res, "\\r\\n")
	if len(res) == 1 {
		log.Info("ParseResponse(): Invalid Redis-server response. Nothing to parse")
		return false
	}

	for _, field := range res {

		kv := strings.Split(field, ":")
		if len(kv) == 2 {
			switch kv[0] {
			case "role":
				R.Role = kv[1]
			case "master_host":
				R.MasterHost = kv[1]
			case "master_port":
				R.MasterPort = kv[1]
			case "slave_repl_offset":
				i, err := strconv.Atoi(kv[1])
				if err == nil {
					R.SyncBytes = int64(i)
				}
			case "master_link_down_since_seconds":
				i, err := strconv.Atoi(kv[1])
				if err == nil {
					R.MasterDownSince = i
				}
			case "master_link_status":
				if kv[1] == "on" {
					R.MasterLinkStatus = true
				} else {
					R.MasterLinkStatus = false
				}
			case "master_last_io_seconds_ago":
				i, err := strconv.Atoi(kv[1])
				if err == nil {
					R.LastUpdated = i
				}
			case "slave_priority":
				i, err := strconv.Atoi(kv[1])
				if err == nil {
					R.Priority = i
				}
			}
		}
	}
	fmt.Printf("R=%v\n", R)
	return true
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

	//log.Infof("CollectStats(%s)=%v", EndPoint, Res.String())
	R.ParseResponse(Res.String())

	R.Client = C
	return &R, nil
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

func GuessMaster(Servers []*Redis) {

	sort.Sort(RedisSlaves(Servers))

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
	GuessMaster(Servers)

	log.Infof("Sorted Servers are %v", Servers)

	log.Infof("Redis-Sentinal-micro Finished")

}
