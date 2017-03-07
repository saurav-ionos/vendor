package Utility

import (
	"os/exec"
	"strings"

	logr "github.com/Sirupsen/logrus"
)

func CreateCommand(command string, arguments []string) *(exec.Cmd) {
	return exec.Command(command, arguments...)
}

func ExecuteCommand(command *(exec.Cmd)) ([]string, error) {
	var output []string

	outBytes, err := command.CombinedOutput()
	outString := string(outBytes)
	output = strings.Split(outString, "\n")

	if err != nil {
		logr.Error("Error executing the command", err.Error(), command.Path, command.Args, outString)
	}
	//cmdReader, err := command.StdoutPipe()
	//if err != nil {
	//	logr.Error("Error getting cmd output pipe", err.Error())
	//}
	//scanner := bufio.NewScanner(cmdReader)
	//go func() {
	//	for scanner.Scan() {
	//		output = append(output, scanner.Text())
	//	}
	//}()

	//err = command.Start()

	//if err != nil {
	//		logr.Error("Error starting the command", err.Error(), command)
	//}

	//err = command.Wait()
	//if err != nil {
	//	logr.Error("Error waiting command exit", err.Error(), command)
	//}

	return output, err
}
