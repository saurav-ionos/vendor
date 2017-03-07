package Utility

import (
	"errors"
	"net"
	"strconv"
	"strings"

	logr "github.com/Sirupsen/logrus"
	Const "github.com/ionosnetworks/qfx_dp/cpewizard/Constants"
	DataObj "github.com/ionosnetworks/qfx_dp/cpewizard/DataObjects"
)

var ipHeaderRangeValues = []int{0, 128, 192, 224, 240, 248, 252, 254, 255}

func IsValidIP(ip string) bool {

	if net.ParseIP(ip) != nil {
		return true
	}
	return false
}

func IsValidSubnet(subnet string) bool {
	validSubnet := true
	if IsValidIP(subnet) {
		ipHeaders := strings.Split(subnet, ".")
		for i, ipHeader := range ipHeaders {
			var nextHeaderInt int
			headerInt, _ := strconv.Atoi(ipHeader)
			if i < len(ipHeaders)-1 {
				nextHeaderInt, _ = strconv.Atoi(ipHeaders[i+1])
			}
			if (!isHeaderValid(headerInt)) || (i != len(ipHeaders)-1 && headerInt < 255 && nextHeaderInt > 0) {
				validSubnet = false
			}
		}
	} else {
		validSubnet = false
	}
	return validSubnet
}

func isHeaderValid(header int) bool {
	isHeaderProper := false
	for _, rangeVal := range ipHeaderRangeValues {
		if header == rangeVal {
			isHeaderProper = true
		}
	}
	return isHeaderProper
}

func IsValidGateWay(gateway string) bool {
	if len(gateway) == 0 {
		return true
	}
	return IsValidIP(gateway)
}

func IsGatewayInRange(interf DataObj.PortConfigInfo) bool {
	if len(interf.GateWay) == 0 {
		return true
	}
	var network net.IPNet
	network.IP = net.ParseIP(interf.IPAddress)
	network.Mask = net.IPMask(net.ParseIP(interf.Subnet).To4())
	return (&network).Contains(net.ParseIP(interf.GateWay))
}

func IsSubnetConflict(ipNet1, ipNet2 *net.IPNet) bool {
	return ipNet1.Contains(ipNet2.IP) || ipNet2.Contains(ipNet1.IP)
}

func CheckIPConflict(interf1, interf2 DataObj.PortConfigInfo) bool {
	if len(interf1.IPAddress) == 0 || len(interf2.IPAddress) == 0 {
		return false
	}
	return interf1.IPAddress == interf2.IPAddress
}

func CheckSubnetConflict(interf1, interf2 DataObj.PortConfigInfo) bool {
	var ipNet1, ipNet2 net.IPNet
	if len(interf1.Subnet) == 0 || len(interf2.Subnet) == 0 {
		return false
	}
	ipNet1.IP = net.ParseIP(interf1.IPAddress)
	ipNet2.IP = net.ParseIP(interf2.IPAddress)
	ipNet1.Mask = net.IPMask(net.ParseIP(interf1.Subnet).To4())
	ipNet2.Mask = net.IPMask(net.ParseIP(interf2.Subnet).To4())
	return IsSubnetConflict(&ipNet1, &ipNet2)
}

func CheckIPAddressConflictForInterface(interf DataObj.PortConfigInfo, currentInterfaces []DataObj.PortConfigInfo) bool {
	var isIpConflict bool
	//interfaces := FileManager.GetInterfaces()
	for _, interfaceObj := range currentInterfaces {
		if interf.PrimaryLan == interfaceObj.PrimaryLan {
			continue
		}
		isIpConflict = CheckIPConflict(interf, interfaceObj)
		if isIpConflict {
			logr.Debug("Ip address conflict for interface with interface", interf, interfaceObj)
			break
		}
	}

	return isIpConflict
}

func CheckSubnetConflictForInterface(interf DataObj.PortConfigInfo, currentInterfaces []DataObj.PortConfigInfo) bool {
	var isSubnetConflict bool

	for _, interfaceObj := range currentInterfaces {
		if interf.PrimaryLan == interfaceObj.PrimaryLan {
			continue
		}
		isSubnetConflict = CheckSubnetConflict(interf, interfaceObj)
		if isSubnetConflict {
			logr.Debug("net range conflict for with interface :", interf, interfaceObj)
			break
		}
	}

	return isSubnetConflict
}

func IsValidInterface(interf DataObj.PortConfigInfo, currentInterfaces []DataObj.PortConfigInfo) (bool, error) {

	if interf.PrimaryLan == Const.NonEditableLogicalEth {
		return false, errors.New(Const.NonEditableEth)
	} else if !IsValidIP(interf.IPAddress) {
		logr.Debug("Invalid ip address for interface", interf)
		return false, errors.New(Const.InvalidIP)
	} else if !IsValidSubnet(interf.Subnet) {
		logr.Debug("Invalid net mask for interface", interf)
		return false, errors.New(Const.InvalidSubnet)
	} else if !(IsValidGateWay(interf.GateWay) && IsGatewayInRange(interf)) {
		logr.Debug("Invalid gateway  for interface", interf)
		return false, errors.New(Const.InvalidGateWay)
	} else if CheckIPAddressConflictForInterface(interf, currentInterfaces) {
		return false, errors.New(Const.IPAlreadyExists)
	} else if CheckSubnetConflictForInterface(interf, currentInterfaces) {
		return false, errors.New(Const.SubnetConflict)
	}
	return true, nil
}

func ValidateAllInterfaces(newInterfaces, oldInterfaces []DataObj.PortConfigInfo) (bool, error) {
	var isValidInterface bool
	var err error
	for _, newInterface := range newInterfaces {

		if newInterface.PrimaryLan == Const.NonEditableLogicalEth {
			isValidInterface = false
			err = errors.New(Const.NonEditableEth)
			logr.Error("Invalid interface", err.Error(), newInterface)
			break
		}

		if newInterface.IPAddressType == Const.IPDhcp {
			isValidInterface = true
			continue
		}
		isValidInterface, err = IsValidInterface(newInterface, oldInterfaces)
		if !isValidInterface {
			logr.Error("Error validating interface", newInterface)
			break
		}
	}
	return isValidInterface, err
}
