package FileManager

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	Const "github.com/ionosnetworks/qfx_dp/cpewizard/Constants"
	DBManager "github.com/ionosnetworks/qfx_dp/cpewizard/DataBase"
	DataObj "github.com/ionosnetworks/qfx_dp/cpewizard/DataObjects"
	Utility "github.com/ionosnetworks/qfx_dp/cpewizard/Utilities"

	logr "github.com/Sirupsen/logrus"
)

func GetInterfaces() []DataObj.PortConfigInfo {
	var interfaceObjs []DataObj.PortConfigInfo
	interfaceFiles, err := ioutil.ReadDir(Const.InterfaceFileFolder)
	if err != nil {
		logr.Error("Error reading interface dir", err.Error())
	}
	multilinkFileData := string(ReadOrCreateMultiLinkFile())
	for _, file := range interfaceFiles {
		fileAbsPath := Const.InterfaceFileFolder + "/" + file.Name()
		fileName := GetFileNameWithoutExtension(file.Name())
		if (fileName == Const.Eth0 || fileName == Const.Eth1 || fileName == Const.Eth2 || fileName == Const.Eth3 || fileName == Const.Eth4 || fileName == Const.Eth5) && (GetExtensionForFile(file.Name()) == "cfg" || GetExtensionForFile(file.Name()) == ".cfg") {
			var ether string
			var phyPort string
			var ipType string
			var isEditable bool
			var ipAddress string
			var subnet string
			var gateway string
			var primaryDns string
			var secondaryDns string
			fileData := string(ReadFile(fileAbsPath))
			lines := strings.Split(string(fileData), "\n")
			for i, line := range lines {

				if len(line) > 0 && string([]rune(line)[0]) == "#" {
					continue
				}
				if strings.Contains(line, "iface") && strings.Contains(line, "eth") {
					words := strings.Fields(line)
					for _, word := range words {
						if strings.Contains(word, "eth") {
							ether = word
							phyPort = Const.EthernetMapping[word]
						}
					}

					if ether == Const.NonEditableLogicalEth {
						isEditable = false
					} else {
						isEditable = true
					}

					if strings.Contains(line, "dhcp") {
						ipType = "dhcp"
						interfaceObjs = append(interfaceObjs, DataObj.PortConfigInfo{PrimaryLan: ether, PhysicalPort: phyPort, IPAddressType: ipType, LanSpeed: Utility.GetLanSpeedForInterface(ether), LinkStatus: Utility.GetLinkStatusForInterface(ether), LanStatus: Utility.GetLanStatusForInterfaceNew(ether), IsEditable: isEditable, IPAddress: Utility.GetDhcpIpForInterface(ether), Subnet: Utility.GetDhcpNetMaskForInterface(ether), MacAddress: Utility.MacAddrForInterface(ether), MultiLinkActive: GetMultiLinkActiveStatus(ether, multilinkFileData)})
					} else if strings.Contains(line, "static") {
						ipType = "static"
						ipLine := lines[i+1]
						netmaskLine := lines[i+2]
						words := strings.Fields(ipLine)
						for _, word := range words {
							if net.ParseIP(word) != nil {
								ipAddress = word //net.ParseIP(word)
							}
						}

						words = strings.Fields(netmaskLine)
						for _, word := range words {
							if net.ParseIP(word) != nil {
								subnet = word //net.ParseIP(word)
							}
						}

						if len(lines)-1 > i+2 {
							gatewayLine := lines[i+3]
							if strings.Contains(gatewayLine, "gateway") {
								words = strings.Fields(gatewayLine)
								for _, word := range words {
									if net.ParseIP(word) != nil {
										gateway = word
									}
								}
							}
						}

						if len(lines)-1 > i+3 {
							dnsServerLine := lines[i+4]
							if strings.Contains(dnsServerLine, "dns-nameservers") {
								words = strings.Fields(dnsServerLine)
								for _, word := range words {
									if net.ParseIP(word) != nil && (len(primaryDns) == 0 || len(primaryDns) == 1) {
										primaryDns = word
									} else if net.ParseIP(word) != nil {
										secondaryDns = word
									}
								}
							}
						}
						interfaceObjs = append(interfaceObjs, DataObj.PortConfigInfo{PrimaryLan: ether, PhysicalPort: phyPort, IPAddressType: ipType, LanSpeed: Utility.GetLanSpeedForInterface(ether), LinkStatus: Utility.GetLinkStatusForInterface(ether), LanStatus: Utility.GetLanStatusForInterfaceNew(ether), IsEditable: isEditable, IPAddress: ipAddress, GateWay: gateway, Subnet: subnet, PrimaryDns: primaryDns, SecondaryDns: secondaryDns, MacAddress: Utility.MacAddrForInterface(ether), MultiLinkActive: GetMultiLinkActiveStatus(ether, multilinkFileData)})
					}

				}

			}
		}
	}
	return interfaceObjs
}

func GetMultiLinkActiveStatus(ethernetPort string, multilinkFileData string) bool {
	activeEthPorts := strings.Split(strings.Split(strings.Replace(multilinkFileData, "\n", "", -1), "=")[1], " ")
	for _, ethPort := range activeEthPorts {
		if ethernetPort == ethPort {
			return true
		}
	}
	return false
}

func ReadOrCreateMultiLinkFile() []byte {
	logr.Info("Reading file", Const.MultiLinkFile)
	dat, err := ioutil.ReadFile(Const.MultiLinkFile)
	if err != nil {
		logr.Error("Error reading file", Const.MultiLinkFile, err.Error())
		logr.Info("Creating new file")
		mappingBytes := ReadFile(Const.EthMapFile)
		if mappingBytes != nil {
			mappingData := Utility.RemoveEmptyStringsFromArray(strings.Split(string(mappingBytes), "\n"))
			ethData := "active = " + strings.Split(mappingData[0], ":")[1]
			logr.Info("Data to written :", ethData)
			ioutil.WriteFile(Const.MultiLinkFile, []byte(ethData), 0644)
		}
	}
	return dat
}

func ReadFile(location string) []byte {
	logr.Info("Reading file", location)
	dat, err := ioutil.ReadFile(location)
	if err != nil {
		logr.Error("Error reading file", location, err.Error())
	}
	return dat
}

func UpdateInterfaceFilesNew(newInterfaces []DataObj.PortConfigInfo) {
	for _, newInterface := range newInterfaces {
		files, _ := ioutil.ReadDir(Const.InterfaceFileFolder)
		for _, file := range files {
			isChange := true
			//var newInterfaceData []string
			fileName := GetFileNameWithoutExtension(file.Name())
			fileAbsPath := Const.InterfaceFileFolder + "/" + file.Name()
			if fileName == newInterface.PrimaryLan {
				fileData := string(ReadFile(fileAbsPath))
				lines := strings.Split(string(fileData), "\n")
				for i, line := range lines {
					if len(line) > 0 && string([]rune(line)[0]) == "#" {
						continue
					}
					if strings.Contains(line, "iface") && strings.Contains(line, "eth") && strings.Contains(line, Const.IPDhcp) && newInterface.IPAddressType == Const.IPDhcp {
						isChange = false
						break
					} else if strings.Contains(line, "iface") && strings.Contains(line, "eth") && strings.Contains(line, Const.IPDhcp) && newInterface.IPAddressType == Const.IPStatic {
						newLine := strings.Replace(line, Const.IPDhcp, Const.IPStatic, -1)
						addrLine := "address " + newInterface.IPAddress
						netMskLine := "netmask " + newInterface.Subnet
						gateWayLine := "gateway " + newInterface.GateWay
						replaceLineInFile(line, newLine, fileAbsPath)
						if len(lines) > i+1 {
							deleteLinesInFileInRange(i+2, len(lines), fileAbsPath)
						}
						appendLineToFile(addrLine, fileAbsPath)
						appendLineToFile(netMskLine, fileAbsPath)
						if net.ParseIP(newInterface.GateWay) != nil {
							appendLineToFile(gateWayLine, fileAbsPath)
						}

						var dnsLine string
						if net.ParseIP(newInterface.PrimaryDns) != nil && net.ParseIP(newInterface.SecondaryDns) != nil {
							dnsLine = "dns-nameservers " + newInterface.PrimaryDns + " " + newInterface.SecondaryDns
						} else if net.ParseIP(newInterface.PrimaryDns) != nil {
							dnsLine = "dns-nameservers " + newInterface.PrimaryDns
						}
						appendLineToFile(dnsLine, fileAbsPath)
						break
					} else if strings.Contains(line, "iface") && strings.Contains(line, "eth") && strings.Contains(line, Const.IPStatic) {
						logr.Debug("new interface is ", newInterface)
						if newInterface.IPAddressType == Const.IPDhcp {
							newifaceLine := strings.Replace(line, Const.IPStatic, Const.IPDhcp, -1)
							replaceLineInFile(line, newifaceLine, fileAbsPath)
							deleteLinesInFileInRange(i+2, len(lines), fileAbsPath)
							//newInterfaceData = append(newInterfaceData, line)
						} else if newInterface.IPAddressType == Const.IPStatic {
							if !isChangeInInf(newInterface) {
								isChange = false
								break
							}
							oldIpLine := lines[i+1]
							var newIpLine string
							words := strings.Fields(oldIpLine)
							for _, word := range words {
								if net.ParseIP(word) != nil {
									newIpLine = strings.Replace(oldIpLine, word, newInterface.IPAddress, -1)
								}
							}
							//newInterfaceData = append(newInterfaceData, ipLine)
							replaceLineInFile(oldIpLine, newIpLine, fileAbsPath)
							oldNetMskLine := lines[i+2]
							var newNetMskLine string
							words = strings.Fields(oldNetMskLine)
							for _, word := range words {
								if net.ParseIP(word) != nil {
									newNetMskLine = strings.Replace(oldNetMskLine, word, newInterface.Subnet, -1)
								}
							}
							//newInterfaceData = append(newInterfaceData, netMskLine)
							replaceLineInFile(oldNetMskLine, newNetMskLine, fileAbsPath)
							var oldGateWayLine string
							var newGateWayLine string
							if net.ParseIP(newInterface.GateWay) != nil {
								if len(lines)-1 > i+2 && (strings.Contains(lines[i+3], "gateway") || strings.Contains(lines[i+3], "GateWay")) {
									oldGateWayLine = lines[i+3]
									words = strings.Fields(oldGateWayLine)
									for _, word := range words {
										if net.ParseIP(word) != nil {
											newGateWayLine = strings.Replace(oldGateWayLine, word, newInterface.GateWay, -1)
											replaceLineInFile(oldGateWayLine, newGateWayLine, fileAbsPath)
										}
									}
								} else {
									newGateWayLine = "gateway " + newInterface.GateWay
									appendLineToFile(newGateWayLine, fileAbsPath)
								}
							}

							var oldDnsLine string
							var newDnsLine string
							if net.ParseIP(newInterface.PrimaryDns) != nil {
								if len(lines)-1 > i+3 && (strings.Contains(lines[i+4], "dns-nameservers") || strings.Contains(lines[i+4], "Dns-NameServers")) {
									oldDnsLine = lines[i+4]
									if net.ParseIP(newInterface.SecondaryDns) != nil {
										newDnsLine = "dns-nameservers " + newInterface.PrimaryDns + " " + newInterface.SecondaryDns
									} else {
										newDnsLine = "dns-nameservers " + newInterface.PrimaryDns
									}
									replaceLineInFile(oldDnsLine, newDnsLine, fileAbsPath)
								} else {
									newDnsLine = "dns-nameservers " + newInterface.PrimaryDns
									if net.ParseIP(newInterface.SecondaryDns) != nil {
										newDnsLine = newDnsLine + " " + newInterface.SecondaryDns
									}
									appendLineToFile(newDnsLine, fileAbsPath)
								}
							} else if len(lines)-1 > i+3 && (strings.Contains(lines[i+4], "dns-nameservers") || strings.Contains(lines[i+4], "Dns-NameServers")) {
								deleteLinesInFileInRange(i+5, len(lines), fileAbsPath)
							}
						}
						break
					}
				}
				if isChange {
					logr.Debug("updating file with interface", fileName, newInterface)
					checkAndCreateModifiedIfaceFile(lines, newInterface, Const.InterfaceFileFolder)
					logr.Debug("updating iface info", newInterface)
					err := Utility.DoIfDownForInterface(newInterface.PrimaryLan)
					if err != nil {
						logr.Error("Error restarting interface", err.Error())
					}
				}
			}
		}
		muiltiLinkFiledata := strings.Replace(string(ReadFile(Const.MultiLinkFile)), "\n", "", -1)
		if newInterface.MultiLinkActive && !strings.Contains(muiltiLinkFiledata, newInterface.PrimaryLan) {
			muiltiLinkFileNewdata := muiltiLinkFiledata + " " + newInterface.PrimaryLan
			updateMultiLinkFile(muiltiLinkFileNewdata + "\n")
		} else if !newInterface.MultiLinkActive && strings.Contains(muiltiLinkFiledata, newInterface.PrimaryLan) {
			muiltiLinkFileNewdata := strings.Replace(muiltiLinkFiledata, " "+newInterface.PrimaryLan, "", -1)
			updateMultiLinkFile(muiltiLinkFileNewdata + "\n")
		}
	}
}

func isChangeInInf(newInf DataObj.PortConfigInfo) bool {
	portConfigs := DBManager.GetPortConfigs()

	for _, portConfig := range portConfigs {
		if portConfig.PrimaryLan == newInf.PrimaryLan {
			return !reflect.DeepEqual(portConfig, newInf)
		}

	}
	return true
}

func updateMultiLinkFile(muiltiLinkFileNewdata string) {
	logr.Info("Updating multilink file with data: ", muiltiLinkFileNewdata)
	erro := ioutil.WriteFile(Const.MultiLinkFile, []byte(muiltiLinkFileNewdata), 0600)
	if erro != nil {
		logr.Error("Error writing to file", Const.MultiLinkFile, erro.Error())
	} else {
		os.OpenFile(Const.MultiLinkModified, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
		cmdName := "/bin/sh"
		cmdArgs := []string{Const.ReStartFwdScript}
		cmd := Utility.CreateCommand(cmdName, cmdArgs)
		out, err := Utility.ExecuteCommand(cmd)
		if err != nil {
			logr.Error("Error Running forwarder script", out, err)
		} else {
			logr.Info("Forwarder script ran and Done Updating multilink file with data: ", muiltiLinkFileNewdata)
		}
	}
}

func CheckMultiLink(newInterfaces []DataObj.PortConfigInfo) (bool, error) {
	var isValidInterface bool
	var err error
	muiltiLinkFiledata := strings.Replace(string(ReadFile(Const.MultiLinkFile)), "\n", "", -1)
	for _, newInterface := range newInterfaces {
		if !newInterface.MultiLinkActive {
			if len(strings.Fields(strings.Split(muiltiLinkFiledata, "=")[1])) == 1 {
				isValidInterface = false
				err = errors.New(Const.MultiLinkFail)
				break
			}
		}
	}
	return isValidInterface, err
}
func replaceLineInFile(oldLine, newLine, filePath string) {
	cmdName := "sudo"
	replaceString := "s/" + oldLine + "/" + newLine + "/g"
	cmdArgs := []string{"sed", "-i", replaceString, filePath}
	cmd := Utility.CreateCommand(cmdName, cmdArgs)
	out, err := Utility.ExecuteCommand(cmd)
	if err != nil {
		logr.Error("Error replacing line with line", oldLine, newLine, filePath, out)
	}
}

func deleteLinesInFileInRange(start int, end int, filePath string) {
	cmdName := "sudo"
	startNumStr := strconv.Itoa(start)
	endNumStr := strconv.Itoa(end)
	delString := startNumStr + "," + endNumStr + "d"
	cmdArgs := []string{"sed", "-i", delString, filePath}
	cmd := Utility.CreateCommand(cmdName, cmdArgs)
	out, err := Utility.ExecuteCommand(cmd)
	if err != nil {
		logr.Error("Error deleting line", filePath, out)
	}
}

func appendLineToFile(line, filePath string) {
	cmdName := "sudo"
	appenString := "$a\\" + line
	cmdArgs := []string{"sed", "-i", "-e", appenString, filePath}
	cmd := Utility.CreateCommand(cmdName, cmdArgs)
	out, err := Utility.ExecuteCommand(cmd)
	if err != nil {
		logr.Error("Error appending line", line, filePath, out)
	}
}
func CreateInterfaceBkpFolder(folder string) {
	err := os.MkdirAll(folder, 0711)
	if err != nil {
		logr.Error("Error creating backup folder", err.Error())
	}
}

func CopyInterfaceFiles(folder string) {
	//if !DBManager.GetCopyFlagForInterfaceFile() {
	files, _ := ioutil.ReadDir(Const.InterfaceFileFolder)
	for _, file := range files {
		logr.Debug("interface backup file is : ", GetFileNameWithoutExtension(file.Name()))
		fileName := GetFileNameWithoutExtension(file.Name())
		if fileName == Const.Eth0 || fileName == Const.Eth1 || fileName == Const.Eth2 || fileName == Const.Eth3 || fileName == Const.Eth4 || fileName == Const.Eth5 {
			bkpFileName := fileName + "bkp " + time.Now().String() + filepath.Ext(file.Name())
			fileSrcPath := Const.InterfaceFileFolder + "/" + file.Name()
			fileDestPath := folder + "/" + bkpFileName
			if !checkIfIfaceFileIsAlreadyBackedup(fileName) {
				CopyFile(fileSrcPath, fileDestPath)
			}
		}
	}
	//DBManager.UpdateFlagForInterfaceFileCopy(true)
	//}
}

func GetFileNameWithoutExtension(fileName string) string {
	extension := filepath.Ext(fileName)
	return fileName[0 : len(fileName)-len(extension)]
}

func GetExtensionForFile(fileName string) string {
	return filepath.Ext(fileName)
}
func CopyFile(fileSource string, fileDestination string) error {
	srcFile, err := os.Open(fileSource)
	if err != nil {
		logr.Error("Error opening file", fileSource, err.Error())
	}
	defer srcFile.Close()
	destFile, err := os.OpenFile(fileDestination, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600) //os.Create(fileDestination)
	if err != nil {
		logr.Error("Error opening file", fileDestination, err.Error())
	}
	if _, err := io.Copy(destFile, srcFile); err != nil {
		destFile.Close()
		return err
	}
	return destFile.Close()
}

func checkIfIfaceFileIsAlreadyBackedup(eth string) bool {
	files, _ := ioutil.ReadDir(Const.InterfaceBkpFolder)
	if len(files) == 0 {
		return false
	}
	bkpFileName := eth + "bkp"
	for _, file := range files {
		logr.Debug("file name and bkp name :", file.Name(), bkpFileName)
		if strings.Contains(file.Name(), bkpFileName) {
			return true
		}
	}
	return false
}

func CheckIfFileExistsAtPath(folder, fileName string) bool {
	files, _ := ioutil.ReadDir(folder)
	for _, file := range files {
		if file.Name() == fileName {
			return true
		}
	}
	return false
}

func CreateSoftLinkForFile(srcFile, destFile string) {
	cmdName := "ln"
	cmdArgs := []string{"-f", "-s", srcFile, destFile}
	cmd := Utility.CreateCommand(cmdName, cmdArgs)
	out, _ := Utility.ExecuteCommand(cmd)
	logr.Debug("soft link output:", out)
}

func ReadConfigFile() DataObj.ConfigObj {
	var config DataObj.ConfigObj
	configData := ReadFile(Const.ConfigFile)
	json.Unmarshal(configData, &config)
	return config
}

func ReadCpeId() string {
	byteCpeId := string(ReadFile(Const.CpeIdFile))
	return string(byteCpeId)
}

func ReadCpeVersion() string {
	var cpeVersion string
	byteCpVersion := ReadFile(Const.CpeVersionFile)
	if byteCpVersion == nil {
		cpeVersion = "version 1.0"
	} else {
		cpeVersion = string(byteCpVersion)
	}
	cpeVersion = strings.Replace(cpeVersion, "\n", "", -1)
	return cpeVersion
}

func CreateCPData(cpstatus bool) {
	cpStatus := "DOWN"
	if cpstatus {
		cpStatus = "UP"
	}
	cpeData := DataObj.CPInfoObj{CPName: "Ionos's CP", Location: "Bangalore", CPeId: ReadCpeId(), CPeStatus: cpStatus, CPeVersion: ReadCpeVersion(), TimeZone: "IST", CloudSeedUptime: "8 hrs", NetworkStatus: "On", Storage: "4 GB"}
	DBManager.CreateCPInfoObj(cpeData)
}

func checkAndCreateModifiedIfaceFile(oldFileData []string, configByUser DataObj.PortConfigInfo, ifaceFolder string) {
	var cmdArgs []string
	cmdName := "sudo"
	filePath := ifaceFolder + "/" + configByUser.PrimaryLan + ".modified"
	for _, line := range oldFileData {
		if strings.Contains(line, "iface") && strings.Contains(line, "eth") && strings.Contains(line, Const.IPDhcp) && configByUser.IPAddressType == Const.IPStatic {
			cmdArgs = []string{"touch", "-a", filePath}
			cmd := Utility.CreateCommand(cmdName, cmdArgs)
			out, _ := Utility.ExecuteCommand(cmd)
			for _, outp := range out {
				logr.Info("creating .modified file", outp)
			}
		} else if strings.Contains(line, "iface") && strings.Contains(line, "eth") && strings.Contains(line, Const.IPStatic) && configByUser.IPAddressType == Const.IPDhcp {
			cmdArgs = []string{"rm", filePath}
			cmd := Utility.CreateCommand(cmdName, cmdArgs)
			out, _ := Utility.ExecuteCommand(cmd)
			for _, outp := range out {
				logr.Info("removing .modified file", outp)
			}
		}
	}
}

func CreateLogTarFile(destTarFile string) {
	logFilesData := ReadFile(Const.LogInfoFile)
	logFiles := strings.Split(string(logFilesData), "\n")
	cmdName := "sudo"
	cmdArgs := []string{"tar", "-czvf", destTarFile}
	for _, srcFile := range logFiles {
		cmdArgs = append(cmdArgs, srcFile)
	}
	cmd := Utility.CreateCommand(cmdName, cmdArgs)
	out, _ := Utility.ExecuteCommand(cmd)
	for _, outp := range out {
		logr.Debug("creating log tar file", outp)
	}

	removeSftLinksExceptLatest()
	timeStamp := time.Now().String()
	//softLink := Const.LogFileSoftLink + " " + timeStamp
	//Const.CurrentLogSoftLink = "ionos.logs " + timeStamp
	timeStamp = strings.Replace(timeStamp, " ", ",", -1)
	softLink := Const.LogFileSoftLink + timeStamp + ".logs"
	Const.CurrentLogSoftLink = "ionos-" + timeStamp + ".logs"
	CreateSoftLinkForFile(destTarFile, softLink)
}

func removeSftLinksExceptLatest() {
	var softLinks []os.FileInfo
	files, err := ioutil.ReadDir(Const.SoftLinkFolder)
	if err != nil {
		logr.Error("Error reading soft links", err.Error())
		return
	}

	for _, file := range files {
		if strings.Contains(file.Name(), "ionos-") {
			softLinks = append(softLinks, file)
		}
	}
	softLinks = sortSoftLinks(softLinks)
	for i := 0; i < len(softLinks)-1; i++ {
		softLinkPath := Const.SoftLinkFolder + "/" + softLinks[i].Name()
		cmdName := "rm"
		cmdArgs := []string{softLinkPath}
		cmd := Utility.CreateCommand(cmdName, cmdArgs)
		out, _ := Utility.ExecuteCommand(cmd)
		logr.Info(out)
	}
}

func sortSoftLinks(files []os.FileInfo) []os.FileInfo {
	for i := 0; i < len(files); i++ {
		//timeStr := strings.Replace(files[i].Name(), "ionos.logs ", "", -1)
		timeStr := strings.Replace(files[i].Name(), "ionos-", "", -1)
		timeStr = strings.Replace(timeStr, ".logs", "", -1)
		timeStr = strings.Replace(timeStr, ",", " ", -1)

		curTime, _ := time.Parse(time.RFC822, timeStr)
		if i != len(files)-1 {
			//nxtTimeStr := strings.Replace(files[i+1].Name(), "ionos.logs ", "", -1)
			nxtTimeStr := strings.Replace(files[i+1].Name(), "ionos-", "", -1)
			nxtTimeStr = strings.Replace(nxtTimeStr, ".logs", "", -1)
			nxtTimeStr = strings.Replace(nxtTimeStr, ",", " ", -1)
			nxtTime, _ := time.Parse(time.RFC822, nxtTimeStr)
			if nxtTime.Before(curTime) {
				tmp := files[i]
				files[i] = files[i+1]
				files[i+1] = tmp
			}
		}
	}
	return files
}

func CreateCmdLogFile() {
	_, err := os.OpenFile(Const.CmdsLogFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)
	if err != nil {
		logr.Error("Error opening file", Const.CmdsLogFile, err.Error())
		return
	}
	var output []string
	cmdName := "ip"
	for _, cmdType := range Const.LogCmds {
		cmdArgs := []string{cmdType, "show"}
		cmd := Utility.CreateCommand(cmdName, cmdArgs)
		out, _ := Utility.ExecuteCommand(cmd)
		cmdOut := cmdName + cmdType + "show" + "\n"
		output = append(output, cmdOut)
		output = append(output, strings.Join(out, "\n"))
		space := "\n" + "\n" + "\n"
		output = append(output, space)
	}
	erro := ioutil.WriteFile(Const.CmdsLogFile, []byte(strings.Join(output, "\n")), 0600)
	if erro != nil {
		logr.Error("Error writing to file", Const.CmdsLogFile, erro.Error())
	}
}
