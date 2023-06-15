package main

import (
    "encoding/json"
    "flag"
    "github.com/go-redis/redis"
    "log"
    "os"
    "redisswitch/kube"
    "strconv"
    "strings"
    "time"
)

type SentinelStruct struct {
    SentinelMasterName string
    SentinelPasswd     string
    SentinelAddrs      []string
}

var (
    SentinelAddresses  = flag.String("sentinel-addr","","sentinel地址连接池，以,分割。如：192.168.10.1:26379,192.168.10.2:26379")
    SentinelMasterName = flag.String("sentinel-name","mymaster","sentinel name")
    SentinelStatusPath = flag.String("sentinel-status","status.json","记录sentinel切换的时间和节点的json文件")
    NameSpaces = flag.String("namespace","entcmd,entuc","redis切换后要重启的namespace名称，以,分割")
    KubeConfig = flag.String("kubeconfig","~/.kube/config","指定Kubeconfig文件路径")
    MasterUrl = flag.String("k8s-master-url","","指定kube-apiservice的地址，如：https://192.168.10.51:6444")
    IntervalTime = flag.String("interval-time","3","探测间隔时间")
)

func getEnv(key, defVal string) string {
    val := os.Getenv(key)
    if val == "" {
        return defVal
    }
    return val
}

func main() {
    // Get the previous configuration of Sentinel
    // fmt.Printf("%s,%s,%s",before_master,current_master,current_sentinel)
    // Determine whether the current value is the same as the previous value
    flag.Parse()
    var config Json
    config = File{path: *SentinelStatusPath}
    before := config.load()
    // Set a state variable，the loop only reads the variable
    beforeMaster := before.Node
    for {
        // Get the current master node
        currentMaster, currentSentinel := getSentinelMaster()
        if beforeMaster != currentMaster {
            current := sentinelStatus{
                LastTime:     time.Now().Format("2006-01-02 15:04:05"),
                Node:         currentMaster,
                SentinelNode: currentSentinel,
            }
            config.update(current)
            beforeMaster = currentMaster
            // restart namespace deployment
            kubectl, err := kube.CreateClient(*MasterUrl,*KubeConfig)
            if err != nil {
                log.Fatal(err)
            }

            // 循环重启namespace
            // for _, namespace := range *NameSpaces {
            //     err = kube.RolloutRestartNamespace(kubectl, namespace)
            //     if err != nil {
            //         log.Fatalln(err)
            //     }
            //     log.Println("Deployment restarted successfully.")
            //     time.Sleep(time.Second * 3)
            // }
            err = kube.RolloutRestartNamespaces(kubectl, *NameSpaces)
            if err != nil {
                log.Fatalln(err)
            }
            log.Printf("Namespace %s Deployment restarted successfully. ", *NameSpaces)
        } else {
            log.Printf("Before: %s Current: %s", beforeMaster, currentMaster)
        }
        IntervalTime,_ := strconv.Atoi(*IntervalTime)
        time.Sleep(time.Second * time.Duration(IntervalTime))
    }
}

// get redis master
func getSentinelMaster() (currentMaster string, currentSentinel string) {
    // Check if the sentinel environment variable is defined
    if *SentinelAddresses == "" {
        log.Fatalln("sentinel-addr parameter unspecified. ")
    }
    if *SentinelMasterName == "" {
        log.Fatalln("sentinel-name parameter unspecified. ")
    }

    sentinel := SentinelStruct{
        SentinelMasterName: *SentinelMasterName,
        SentinelAddrs:      []string(strings.Split(*SentinelAddresses, ",")),
    }

    for _, SentinelAddr := range sentinel.SentinelAddrs {
        sentinelCli := redis.NewSentinelClient(&redis.Options{
            Addr:        SentinelAddr,
            DialTimeout: time.Second * 5,
        })
        masterAddr, err := sentinelCli.GetMasterAddrByName(sentinel.SentinelMasterName).Result()
        if err != nil {
            log.Printf("Warn: Sentinel:%s . %s", SentinelAddr, err)
            continue
        }
        sentinelErr := sentinelCli.Close()
        if sentinelErr != nil {
            log.Fatalln("The redis connection could not be closed properly")
        }
        if masterAddr[0] != "" {
            return masterAddr[0], SentinelAddr
        }
    }
    log.Fatalln("The master node cannot be found, please check the redis sentinel configuration. ")
    return
}

type Json interface {
    load() *sentinelStatus
    update(sentinelStatus)
}
type File struct {
    path string
}

type sentinelStatus struct {
    LastTime     string `json:"last_time"`
    Node         string `json:"node"`
    SentinelNode string `json:"sentinel_node"`
}

func (name File) load() *sentinelStatus {
    // var status sentinelStatus
    // file, err := ioutil.ReadFile(name.path)
    file, err := os.ReadFile(name.path)
    if err != nil {
        log.Fatalf("Some error occured while reading file. Error: %s", err)
    }
    status := &sentinelStatus{}
    err = json.Unmarshal(file, status)
    if err != nil {
        log.Fatalf("Error occured during unmarshaling. Error: %s", err.Error())
    }
    return status
}

func (name File) update(result sentinelStatus) {
    data, err := json.Marshal(result)
    if err != nil {
        log.Fatalln(err)
    }
    e := os.WriteFile(name.path, data, 0755)
    if e != nil {
        log.Fatalf("Error writing to result file: %s.", name.path)
    }
    log.Printf("A change has been detected on the master node, and the {%s} has been updated", name.path)
}
