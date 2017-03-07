package Utility

import (
	"bufio"
	"fmt"
	//"ics/ica-dataplane/cp"
	"net"
	"strconv"
	"strings"
	"time"

	logr "github.com/Sirupsen/logrus"
	//"github.com/ionosnetworks.com/ics/ica-dataplane/cp"
	Const "github.com/ionosnetworks/qfx_dp/cpewizard/Constants"
	DataObj "github.com/ionosnetworks/qfx_dp/cpewizard/DataObjects"
)

var (
//WizCpChannel chan cp.CpWizMsg
//CpWizChannel chan cp.CpWizMsg
)

//func check(e error) {
//	if e != nil {
//		logr.Warning("error", e)
//	}
//}

//GetWanStatusForInterfaces for cpv1 4 interfaces and cpv2 6 interfaces
func GetWanStatusForInterfaces(currentCPEInfo DataObj.CPInfoObj) interface{} {
	var wanStatusForInterfaces interface{}
	if currentCPEInfo.CPeVersion == "1f" {
		wanStatusForInterfaces = DataObj.WanStatusInfoV2{
			Eth0: GetWanStatusForInterface(Const.Eth0, Const.PingResponseCount, Const.PingCheckIP),
			Eth1: GetWanStatusForInterface(Const.Eth1, Const.PingResponseCount, Const.PingCheckIP),
			Eth2: GetWanStatusForInterface(Const.Eth2, Const.PingResponseCount, Const.PingCheckIP),
			Eth3: GetWanStatusForInterface(Const.Eth3, Const.PingResponseCount, Const.PingCheckIP),
			Eth4: GetWanStatusForInterface(Const.Eth4, Const.PingResponseCount, Const.PingCheckIP),
			Eth5: GetWanStatusForInterface(Const.Eth5, Const.PingResponseCount, Const.PingCheckIP),
		}
	} else {
		wanStatusForInterfaces = DataObj.WanStatusInfoV1{
			Eth0: GetWanStatusForInterface(Const.Eth0, Const.PingResponseCount, Const.PingCheckIP),
			Eth1: GetWanStatusForInterface(Const.Eth1, Const.PingResponseCount, Const.PingCheckIP),
			Eth2: GetWanStatusForInterface(Const.Eth2, Const.PingResponseCount, Const.PingCheckIP),
			Eth3: GetWanStatusForInterface(Const.Eth3, Const.PingResponseCount, Const.PingCheckIP),
		}
	}
	return wanStatusForInterfaces
}

//CheckInternetConnectivity using URl ex: golang.org/google.com
func CheckInternetConnectivity(netwrkAddr string) bool {
	conn, err := net.Dial("tcp", netwrkAddr+":"+Const.NetWorkPort)
	if err != nil {
		logr.Error("Error connecting to internet", netwrkAddr, err.Error())
	}
	if conn != nil {
		fmt.Fprintf(conn, "GET / HTTP/1.0\r\n\r\n")
		status, err := bufio.NewReader(conn).ReadString('\n')
		if err != nil {
			logr.Error("Error reading connection buffer", err.Error())
		}
		logr.Debug("connect status is :", status)
		return true
	} else {
		logr.Debug("Not Connected to internet")
		return false
	}
}

//CheckInternetConnectivity with ip ping ex: ping 8.8.8.8
func CheckInternetConnectivityWithPing(netwrkAddr string) bool {
	var dataReceiveCount, hostNotReachableCount int
	dataString := "bytes from " + netwrkAddr
	errorString := "Destination Host Unreachable"
	cmdName := "ping"
	cmdArgs := []string{"-c", Const.PingResponseCount, netwrkAddr}
	cmd := CreateCommand(cmdName, cmdArgs)
	out, _ := ExecuteCommand(cmd)
	for _, outp := range out {
		if strings.Contains(outp, dataString) {
			dataReceiveCount++
		} else if strings.Contains(outp, errorString) || strings.Contains(outp, Const.PingNetworkUnreachable) {
			hostNotReachableCount++
		}
	}

	if hostNotReachableCount == 0 || dataReceiveCount > hostNotReachableCount {
		return true
	} else if dataReceiveCount == 0 {
		return false
	}

	return false
}

func GetDhcpIpForInterface(eth string) string {
	var ipAddr string
	cmdName := "ifconfig"
	cmdArgs := []string{eth}
	cmd := CreateCommand(cmdName, cmdArgs) //exec.Command(cmdName, cmdArgs...)
	out, _ := ExecuteCommand(cmd)
	for _, outp := range out {
		if strings.Contains(outp, "addr:") && strings.Contains(outp, "Bcast:") && strings.Contains(outp, "Mask:") {
			words := strings.Fields(outp)
			for _, word := range words {
				if strings.Contains(word, "addr:") {
					ipAddr = strings.Replace(word, "addr:", "", -1)
				}
			}
		}
	}
	return ipAddr
}

func GetDhcpNetMaskForInterface(eth string) string {
	var netMask string
	cmdName := "ifconfig"
	cmdArgs := []string{eth}
	cmd := CreateCommand(cmdName, cmdArgs) //exec.Command(cmdName, cmdArgs...)
	out, _ := ExecuteCommand(cmd)
	for _, outp := range out {
		if strings.Contains(outp, "addr:") && strings.Contains(outp, "Bcast:") && strings.Contains(outp, "Mask:") {
			words := strings.Fields(outp)
			for _, word := range words {
				if strings.Contains(word, "Mask:") {
					netMask = strings.Replace(word, "Mask:", "", -1)
				}
			}
		}
	}
	return netMask
}

func MacAddrForInterface(eth string) string {
	var macAddr string
	cmdName := "ifconfig"
	cmdArgs := []string{eth}
	cmd := CreateCommand(cmdName, cmdArgs) //exec.Command(cmdName, cmdArgs...)
	out, _ := ExecuteCommand(cmd)
	for _, outp := range out {
		if strings.Contains(outp, "HWaddr") {
			words := strings.Fields(outp)
			for _, word := range words {
				if _, err := net.ParseMAC(word); err == nil {
					macAddr = word
				}
			}
		}
	}
	return macAddr
}

//GetLinkStatusForInterface using command line ex: ethtool eth0
func GetLinkStatusForInterface(eth string) bool {
	cmdName := "ethtool"
	cmdArgs := []string{eth}
	cmd := CreateCommand(cmdName, cmdArgs) //exec.Command(cmdName, cmdArgs...)
	out, _ := ExecuteCommand(cmd)
	for _, outp := range out {
		if strings.Contains(outp, "Link detected") && strings.Contains(outp, "yes") {
			return true
		} else if strings.Contains(outp, "Link detected") && strings.Contains(outp, "no") {
			return false
		}
	}
	return false
}

//DoIfDownForInterface after updating file in /etc/network/interfaces.d
func DoIfDownForInterface(eth string) error {
	logr.Debug("Doing ifdown iface", eth)
	cmdName := "sudo" //"ifconfig"
	cmdArgs := []string{"ifconfig", eth, "down"}
	cmd := CreateCommand(cmdName, cmdArgs)
	out, _ := ExecuteCommand(cmd)
	for _, outp := range out {
		logr.Debug("ifdown output for interface :", outp, eth)
	}
	return nil
}

//GetLanStatusForInterface using ip command, ex: ip link show eth
func GetLanStatusForInterface(eth, object, cmdType string) bool {
	cmdName := "ip"
	cmdArgs := []string{object, cmdType, eth}
	cmd := CreateCommand(cmdName, cmdArgs) //exec.Command(cmdName, cmdArgs...)
	out, _ := ExecuteCommand(cmd)
	for _, outp := range out {
		if strings.Contains(outp, Const.LanStatusUPString) {
			return true
		} else if strings.Contains(outp, Const.LanStatusDownString) {
			return false
		}
	}
	return false
}

func GetLanStatusForInterfaceNew(eth string) bool {
	if !GetLinkStatusForInterface(eth) {
		return false
	}

	gateWay := GetGateWayForInterface(eth)

	if len(gateWay) < 1 {
		return false
	}

	lanSta := GetWanStatusForInterface(eth, "2", gateWay)

	if lanSta == "UP" {
		return true
	}
	return false //GetWanStatusForInterface(eth, "2", gateWay)
}

func GetGateWayForInterface(eth string) string {
	var gateWay string
	i, _ := strconv.Atoi(strings.Replace(eth, "eth", "", -1))
	tblnum := strconv.Itoa(i + 1)
	cmdName := "ip"
	cmdArgs := []string{"route", "show", "table", tblnum}
	cmd := CreateCommand(cmdName, cmdArgs)
	out, _ := ExecuteCommand(cmd)
	if len(out) == 0 {
		return ""
	}

	for _, outp := range out {
		if strings.Contains(outp, "default via") {
			words := strings.Fields(outp)
			for _, word := range words {
				if net.ParseIP(word) != nil {
					gateWay = word //net.ParseIP(word)
				}
			}

		}
	}
	return gateWay
}

//GetWanStatusForInterface using ping ex: ping -I eth0 8.8.8.8
func GetWanStatusForInterface(eth string, responsecount string, ipAddr string) string {
	var dataReceiveCount int //hostNotReachableCount int
	var wanStatus string
	dataString := "bytes from " + ipAddr
	//errorString := "Destination Host Unreachable"
	cmdName := "ping"
	cmdArgs := []string{"-I", eth, "-c", responsecount, ipAddr}
	cmd := CreateCommand(cmdName, cmdArgs)
	out, _ := ExecuteCommand(cmd)
	for _, outp := range out {
		if strings.Contains(outp, dataString) {
			dataReceiveCount++
		}
		//else if strings.Contains(outp, errorString) || strings.Contains(outp, Const.PingNetworkUnreachable) {
		//	hostNotReachableCount++
		//}
	}

	count, _ := strconv.Atoi(responsecount)
	if dataReceiveCount >= count/2 {
		wanStatus = "UP"
	} else {
		wanStatus = "DOWN"
	}

	//if hostNotReachableCount == 0 || dataReceiveCount > hostNotReachableCount {
	//	wanStatus = "UP"
	//} else if dataReceiveCount == 0 {
	//	wanStatus = "DOWN"
	//}

	return wanStatus
}

//GetLanSpeedForInterface using ethtool command ex: ethtool eth0
func GetLanSpeedForInterface(eth string) string {
	logr.Debug("lan speed for eth :", eth)
	cmdName := "ethtool"
	cmdArgs := []string{eth}
	cmd := CreateCommand(cmdName, cmdArgs) //exec.Command(cmdName, cmdArgs...)
	out, _ := ExecuteCommand(cmd)
	for _, outp := range out {
		if strings.Contains(outp, "Speed:") {
			words := strings.Fields(outp)
			return words[1]
		}
	}
	return ""
}

//GetInterfaces using golang net packages instead of reading from /etc/network/interfaces.d
func GetInterfaces() []DataObj.PortConfigInfo {
	var lanInterfaces []DataObj.PortConfigInfo
	interfaces, err := net.Interfaces()
	if err != nil {
		logr.Error("Error getting interfaces", err.Error())
	}
	for _, interf := range interfaces {
		if strings.Contains(interf.Name, "en") || strings.Contains(interf.Name, "eth") {
			addrs, err := interf.Addrs()
			if err != nil {
				logr.Error("Error reading interface address", interf, err.Error())
			}
			var ip net.IP
			for _, addr := range addrs {
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				ip = ip.To4()
			}
			lanInterfaces = append(lanInterfaces, DataObj.PortConfigInfo{})
		}
	}
	return nil
}

//setTimeOut general timeout
func setTimeOut(timeChan chan bool, duration int) {
	time.Sleep(time.Duration(duration) * time.Second)
	timeChan <- true
}

//GetIc2Connectivity connectivity with ionos controller
func GetIc2Connectivity() bool {
	//var CpWizChnMsg cp.CpWizMsg
	//WizCpChannel <- CpWizChnMsg

	//timeout := make(chan bool, 1)
	//go setTimeOut(timeout, 3)

	//select {
	//case connRespon := <-CpWizChannel:
	//	logr.Debug("IC2 connection response", connRespon)
	//	return connRespon.IC2Conn
	//case <-timeout:
	//	logr.Debug("IC2 connection timedout")
	//	return false
	//}
	//connRespon := <-CpWizChannel
	//return connRespon.IC2Conn
	return false
}

func DnsResolution(addr string) bool {
	hostNames, err := net.LookupIP(addr)
	if err != nil {
		logr.Error("Error resolving dns for address", addr, err.Error())
		return false
	}

	logr.Debug("Dns for address", addr, hostNames)
	return true
}
