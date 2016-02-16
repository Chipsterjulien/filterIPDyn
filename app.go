package main

import (
	"strings"
	"strconv"
	"os/exec"
	"os"
	"time"
	"net"
	"fmt"
	
	"github.com/op/go-logging"
	"github.com/spf13/viper"
)

var (
	C Config
)

type Config struct {
	IpList[] struct {
		isDone bool
		IsHost bool
		IP string
		RealIP string
		PortBegin int
		PortEnd int
		Protocol string
	}
}

func browseIP() {
	log := logging.MustGetLogger("log")

	for i := 0; i < len(C.IpList); i++ {
		if C.IpList[i].PortBegin > C.IpList[i].PortEnd {
			log.Warning("Port end (\"%d\") is smaller than (\"%d\") !", C.IpList[i].PortEnd, C.IpList[i].PortBegin)
			continue
		}

		if !C.IpList[i].isDone || C.IpList[i].IsHost {
			ipList, err := net.LookupHost(C.IpList[i].IP)
			if err != nil {
				log.Warning("Unable to convert \"%s\" to an IP: %v", C.IpList[i].IP, err)
				continue
			}
			if ipList[0] != C.IpList[i].RealIP {
				if C.IpList[i].RealIP == "" {
					for j := C.IpList[i].PortBegin; j <= C.IpList[i].PortEnd; j++ {
						cmdStr := generateStr(C.IpList[i].Protocol, j, ipList[0], "I")
						execCmd(cmdStr)
					}
				} else {
					for _, c := range []string{"D", "I"} {
						for j := C.IpList[i].PortBegin; j <= C.IpList[i].PortEnd; j++ {
							cmdStr := generateStr(C.IpList[i].Protocol, j, ipList[0], c)
							execCmd(cmdStr)
						}
					}
				}
				C.IpList[i].RealIP = ipList[0]
			}
			C.IpList[i].isDone = true
		}
	}
}

func checkHost() {
	for i := 0; i < len(C.IpList); i++ {
		if !isIp(C.IpList[i].IP) {
			C.IpList[i].IsHost = true
		}
	}
}

func execCmd(cmdStr string) {
	log := logging.MustGetLogger("log")
	log.Debug("cmdStr: %s", cmdStr)

	cmd := exec.Command("/sbin/sh", "-c", cmdStr)
	if err := cmd.Start(); err != nil {
		log.Critical("Unable to exec cmd: %v", err)
		return
	}
	if err := cmd.Wait(); err != nil {
		log.Critical("Some error while waiting: %v", err)
		return
	}
}

func generateStr(protocol string, port int, ip string, di string) string {
	return fmt.Sprintf("iptables -%s INPUT -p %s --dport %d -s %s -j ACCEPT", di, protocol, port, ip)
}

func getConfig() {
	log := logging.MustGetLogger("log")

	if err := viper.Marshal(&C); err != nil {
		log.Critical("Unable to translate config file: %v", err)
		os.Exit(1)
	}	
}

func isIp(ip string) bool {
	ipSplitted := strings.Split(ip, ".")

	for _, part := range ipSplitted {
		if _, err := strconv.Atoi(part); err != nil {
			return false
		}
	}

	return true
}

func startApp() {
	log := logging.MustGetLogger("log")

	getConfig()
	log.Debug("Config before check host: %v", C)
	checkHost()
	log.Debug("Config after check host: %v", C)

	for {
		u := time.Now()
		browseIP()
		now := time.Now()
		log.Debug("Diff: %v", now.Sub(u))
		time.Sleep(time.Duration(viper.GetInt("default.refresh")) * time.Second)
	}
}

func main() {
	// confPath := "/etc/filteripdyn/"
	// confFilename := "filteripdyn"
	// logFilename := "/var/log/filteripdyn/error.log"

	confPath := "cfg"
	confFilename := "filteripdyn"
	logFilename := "error.log"

	fd := initLogging(&logFilename)
	defer fd.Close()

	loadConfig(&confPath, &confFilename)

	startApp()
}