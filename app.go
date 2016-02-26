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
		isHost bool
		IP string
		RealIP string
		Protocol string
		PortList []string
	}
}

func browseDynIP() {
	log := logging.MustGetLogger("log")

	strList := []string{}
	for i := 0; i < len(C.IpList); i++ {
		if !C.IpList[i].isHost {
			continue
		}
		
		ipList, err := net.LookupHost(C.IpList[i].IP)
		if err != nil {
			log.Warning(fmt.Sprintf("Unable to convert \"%s\" to an IP: %v", C.IpList[i].IP, err))
			continue
		}
		log.Debug(fmt.Sprintf("Config IP: %s", C.IpList[i].IP))
		log.Debug(fmt.Sprintf("Real IP: %s", C.IpList[i].RealIP))

		if ipList[0] != C.IpList[i].RealIP {
			for _, port := range C.IpList[i].PortList {
				portList := strings.Split(port, ":")
				
				begin, _ := strconv.Atoi(portList[0])
				log.Debug(fmt.Sprintf("Begin port: %d", begin))

				end, _ := strconv.Atoi(portList[1])
				log.Debug(fmt.Sprintf("End port: %d", end))
				
				for j := begin; j <= end; j++ {
					if C.IpList[i].RealIP != "" {
						strList = append(strList, generateStr(C.IpList[i].Protocol, j, C.IpList[i].RealIP, "D"))
					}
					strList = append(strList, generateStr(C.IpList[i].Protocol, j, ipList[0], "I"))
				}
			}
			C.IpList[i].RealIP = ipList[0]
		}
	}

	for _, str := range strList {
		log.Debug(fmt.Sprintf("String cmd: %s", str))
		execCmd(&str)
	}
}

func checkConfig() {
	log := logging.MustGetLogger("log")

	for i := 0; i < len(C.IpList); i++ {
		if C.IpList[i].IP == "" {
			log.Critical("IP must not be an empty string !")
			os.Exit(1)
		}
		if !strings.Contains(C.IpList[i].Protocol, "tcp") && !strings.Contains(C.IpList[i].Protocol, "udp") {
			log.Critical(fmt.Sprintf("For \"%s\", protocol must be \"tcp\" or \"udp\" and not %s",
				C.IpList[i].IP, C.IpList[i].Protocol))
			os.Exit(1)
		}
		for _, port := range C.IpList[i].PortList {
			portList := strings.Split(port, ":")
			if len(portList) != 2 {
				log.Critical(fmt.Sprintf("For \"%s\", the list of ports is not well informed !", C.IpList[i].IP))
				os.Exit(1)
			}
			if _, err := strconv.Atoi(portList[0]); err != nil {
				log.Debug(fmt.Sprintf("For \"%s\", unable to convert \"%s\" into an integer: %v",
					C.IpList[i].IP, portList[0], err))
				os.Exit(1)
			}
			if _, err := strconv.Atoi(portList[1]); err != nil {
				log.Debug(fmt.Sprintf("For \"%s\", unable to convert \"%s\" into an integer: %v",
					C.IpList[i].IP, portList[1], err))
				os.Exit(1)
			}
		}
	}
}

func checkHost() {
	for i := 0; i < len(C.IpList); i++ {
		if !isIp(C.IpList[i].IP) {
			C.IpList[i].isHost = true
		}
	}
}

func execCmd(cmdStr *string) {
	log := logging.MustGetLogger("log")
	log.Debug("cmdStr:", *cmdStr)

	cmd := exec.Command("/sbin/sh", "-c", *cmdStr)
	if err := cmd.Start(); err != nil {
		log.Critical("Unable to exec cmd:", err)
		return
	}
	if err := cmd.Wait(); err != nil {
		log.Critical("Some error while waiting:", err)
		return
	}
}

func generateStr(protocol string, port int, ip string, di string) string {
	return fmt.Sprintf("iptables -%s INPUT -p %s --dport %d -s %s -j ACCEPT", di, protocol, port, ip)
}

func getConfig() {
	log := logging.MustGetLogger("log")

	if err := viper.Unmarshal(&C); err != nil {
		log.Critical("Unable to translate config file:", err)
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

func loadStaticIP() {
	log := logging.MustGetLogger("log")

	strList := []string{}
	for _, list := range C.IpList {
		if !list.isHost {
			log.Debug(list.IP)
			for _, port := range list.PortList {
				portList := strings.Split(port, ":")

				begin, _ := strconv.Atoi(portList[0])
				log.Debug(fmt.Sprintf("Begin port: %d", begin))

				end, _ := strconv.Atoi(portList[1])
				log.Debug(fmt.Sprintf("End port: %d", end))

				for i := begin; i <= end; i++ {
					strList = append(strList, generateStr(list.Protocol, i, list.IP, "I"))
				}
			}
		}
	}

	for _, str := range strList {
		log.Debug(fmt.Sprintf("String cmd: %s", str))
		execCmd(&str)
	}
}

func startApp() {
	log := logging.MustGetLogger("log")

	getConfig()
	log.Debug("Config before check host:", C)
	checkConfig()
	checkHost()
	log.Debug("Config after check host:", C)
	loadStaticIP()

	for {
		browseDynIP()
		time.Sleep(time.Duration(viper.GetInt("default.refresh")) * time.Second)
	}
}

func main() {
	confPath := "/etc/filteripdyn/"
	confFilename := "filteripdyn"
	logFilename := "/var/log/filteripdyn/error.log"

	// confPath := "cfg"
	// confFilename := "filteripdyn"
	// logFilename := "error.log"

	fd := initLogging(&logFilename)
	defer fd.Close()

	loadConfig(&confPath, &confFilename)

	startApp()
}