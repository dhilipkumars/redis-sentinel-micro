package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"encoding/json"
	log "github.com/golang/glog"
	"github.com/mediocregopher/radix.v2/redis"
	"strconv"
)

const (
	COLLECTSTATS_INVALID_INPUT        = "Invalid Endpoint"
	COLLECTSTATS_SERVER_NOT_REACHABLE = "Redis Server Not Reachable"
	REDIS_ROLE_MASTER                 = "master"
	REDIS_ROLE_SLAVE                  = "slave"
)

type Redis struct {
	EndPoint         string        //End point of the redis-server
	Role             string        //Role of this redis-sever Master or Slave
	LastUpdated      int           // When did last sync happened seconds
	MasterDownSince  int           //Since how long the master is not reachable?
	SyncBytes        int64         //How much of data did this sync
	MasterHost       string        //Masters ip addres
	MasterPort       string        //Master port
	Priority         int           //Slave priority
	MasterLinkStatus bool          //true means up false mean down
	Client           *redis.Client //Redis client
}

type RedisSlaves []*Redis

func (rs RedisSlaves) Len() int { return len(rs) }
func (rs RedisSlaves) Swap(i int, j int) {
	var tmp *Redis
	tmp = rs[i]
	rs[i] = rs[j]
	rs[j] = tmp
}

func (rs RedisSlaves) Less(i int, j int) bool {

	//Choose the slave with least priority
	if rs[i].Priority != 0 && rs[j].Priority != 0 {
		if rs[i].Priority < rs[j].Priority {
			return true
		}
	}

	//Choose the slave with maximum SyncBytes
	if rs[i].SyncBytes > rs[j].SyncBytes {
		return true
	}

	//Choose the slave with least Updated time
	if rs[i].LastUpdated < rs[j].LastUpdated {
		return true
	}

	return false
}

func (R *Redis) ParseResponse(Res string) bool {

	res := strings.Split(Res, "\\r\\n")
	if len(res) == 1 {
		log.Errorf("ParseResponse(): Invalid Redis-server response. Nothing to parse")
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
			case "master_repl_offset":
				i, err := strconv.Atoi(kv[1])
				if err == nil && i > 0 {
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

	//Check if the supplied EP is valid
	if len(strings.Split(EndPoint, ":")) != 2 {
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

	R.EndPoint = EndPoint
	R.Client = C
	return &R, nil
}

//CollectStatsAll Contact all the redis containers and collect statistics required to perform a Slave Promotion
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

//FindNxtMaster This function will return next suitable master if there is such a situation otherwise it simply returns nil, for instance if the supplied list of containrs already form a proper master-slave cluster then it will leave the setup intact.
func FindNxtMaster(Servers []*Redis) *Redis {

	var Slaves []*Redis

	//Check if Master is already there
	var isMasterAvailable bool
	var availableMaster *Redis
	var availableMasterHits int

	//Loop through all the servers and find of if there is already a master
	for _, rs := range Servers {
		//TODO: There might be a situation where there are multiple mis-configured masters, should handle that later
		if strings.Contains(rs.Role, REDIS_ROLE_MASTER) {

			isMasterAvailable = true
			availableMaster = rs
			break
		}

	}

	for _, rs := range Servers {
		if isMasterAvailable {
			if rs.EndPoint != availableMaster.EndPoint {
				Slaves = append(Slaves, rs)
				log.Infof("RSMaster_EP=%s available MasterEP=%v", rs.MasterHost+":"+rs.MasterPort, availableMaster.EndPoint)
				if rs.MasterHost+":"+rs.MasterPort == availableMaster.EndPoint {
					availableMasterHits++
				}
			}
		} else {
			Slaves = append(Slaves, rs)
		}
	}
	//If master is available check if its already pouparly configured
	if isMasterAvailable {

		if availableMaster.SyncBytes > 0 && availableMasterHits == len(Slaves) {

			//Looks like the master is active and configured properly
			log.Warningf("The redis master is already configured, dont do anything SyncBytes=%v availableMasterHits=%v len(Slaves)=%v", availableMaster.SyncBytes, availableMasterHits, len(Slaves))
			return nil
		} else {
			log.Warningf("A Redis master is found, but misconfigured, considering it as a slave")
			Slaves = append(Slaves, availableMaster)
		}
	}

	if len(Slaves) == 0 {
		return nil
	}

	//Sort the slaves according the parameters
	sort.Sort(RedisSlaves(Slaves))

	//return the selected slaves
	return Slaves[0]
}

func PromoteASlave(NewMaster *Redis, Servers []*Redis) bool {

	result := true

	//Make the slave as the master first
	resp := NewMaster.Client.Cmd("SLAVEOF", "NO", "ONE").String()
	if !strings.Contains(resp, "OK") {
		log.Errorf("Unable to make the slave as master response=%v", resp)
		return false
	}

	hostPort := strings.Split(NewMaster.EndPoint, ":")
	NewMaster.MasterHost = hostPort[0]
	NewMaster.MasterPort = hostPort[1]

	for _, rs := range Servers {

		if rs.EndPoint == NewMaster.EndPoint {
			continue
		}
		resp = rs.Client.Cmd("SLAVEOF", NewMaster.MasterHost, NewMaster.MasterPort).String()
		if !strings.Contains(resp, "OK") {
			log.Errorf("Unable to make the slave point to new master response=%v", resp)
			return false
		}
	}
	return result
}

func PrintServers(message string, Servers []*Redis) {
	var result string
	result = fmt.Sprintf("PrintServers()\n")
	result += fmt.Sprintf("******%s******\n", message)
	r, _ := json.MarshalIndent(Servers, "", "  ")
	result += string(r)
	result += fmt.Sprintf("*****************\n")
	log.V(2).Info(result)
}

func main() {

	var ServersEndPoint []string

	//Parse command line arguments
	for _, arg := range os.Args[1:] {
		if strings.Contains(arg, ":") {
			ServersEndPoint = append(ServersEndPoint, arg)
		}
	}

	flag.Parse()
	log.Infof("Supplied Arguments are %v", ServersEndPoint)

	//Collect stats on all the redis-servers supplied
	Servers := CollectStatsAll(ServersEndPoint)

	PrintServers("Supplied Servers", Servers)

	//Does it really need a master
	NewMaster := FindNxtMaster(Servers)
	if NewMaster == nil {
		log.Errorf("Redis Instance does'nt need a Slave Promotion")
		os.Exit(1)
	}
	log.Infof("New Master=%v", *NewMaster)

	//Now we have a potential master
	if !PromoteASlave(NewMaster, Servers) {

		PrintServers("In-consistantly configured", Servers)
		log.Errorf("Error occured in Slave Promotion")
		os.Exit(1)
	}
	log.Infof("New Master is %v, All the slaves are re-configured to replicate from this", NewMaster.EndPoint)
	PrintServers("Processed Servers", Servers)

	//
	log.Infof("Redis-Sentinal-micro Finished")

}