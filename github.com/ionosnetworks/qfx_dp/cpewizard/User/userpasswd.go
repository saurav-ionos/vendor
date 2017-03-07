package User

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"

	logr "github.com/Sirupsen/logrus"
)

type User struct {
	UserName string `json:username`
	Password string `json:password`
}

type UserJson struct {
	Users []User `json:users`
}

const (
	alphaBytes        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	alphaNumeric      = "1234567890" + alphaBytes
	letterIdxBits     = 6
	letterIdxMask     = 1<<letterIdxBits - 1
	cpeIdFileLoc      = "/etc/ionos-cpeid.conf"
	userConfigFileLoc = "/var/ionos/userconf"
)

func main1() {
	var opt string

	if len(os.Args) < 2 {
		fmt.Println("Not enough Arguments")
		os.Exit(1)
	} else {
		opt = os.Args[1]
	}

	switch opt {

	case "":
		break
	case "-c":

		if len(os.Args) > 3 {
			updateUser(os.Args[2], os.Args[3], opt)
		} else if len(os.Args) > 2 {
			updateUser(os.Args[2], "", opt)
		} else {
			fmt.Println("Please provide an email id")
		}

	case "-g":

		if len(os.Args) > 3 {
			generateUser(os.Args[2], os.Args[3])
		} else if len(os.Args) > 2 {
			generateUser(os.Args[2], "")
		} else {
			generateUser("", "")
		}

	case "-u":
		if len(os.Args) > 3 {
			updateUser(os.Args[2], os.Args[3], opt)
		} else if len(os.Args) > 2 {
			fmt.Println("please provide password")
		} else {
			fmt.Println("Please provide user details")
		}

	case "-d":
		if len(os.Args) > 2 {
			deleteUser(os.Args[2])
		} else {
			fmt.Println("please provide user details")
		}
	default:
		fmt.Println("Invalid option", opt)
	}
}

func isValidMail(mailstr string) bool {
	_, err := mail.ParseAddress(mailstr)
	if err != nil {
		return false
	}
	return true
}

func updateUser(username, pass, option string) {
	var userExists bool
	var passwd string
	var fileJson UserJson
	if !isValidMail(username) {
		fmt.Println("Please provide a valid email id")
		os.Exit(1)
	}

	if pass == "" || pass == " " {
		passwd = secureRandomAlphaNumericString(10)
	} else {
		passwd = pass
	}

	if _, err := os.Stat(userConfigFileLoc); os.IsNotExist(err) {
		var newUsers []User
		newUser := User{UserName: username, Password: passwd}
		newUsers = append(newUsers, newUser)
		fileJson = UserJson{Users: newUsers}
	} else {
		fileJson, _ = GetUsers()
		currentUsers := fileJson.Users
		for i := 0; i < len(currentUsers); i++ {
			currUser := currentUsers[i]
			if currUser.UserName == username && option == "-c" {
				userExists = true
				break
			} else if currUser.UserName == username {
				currUser.Password = passwd
				currentUsers[i] = currUser
			}
		}

		if userExists && option == "-c" {
			fmt.Println("User Already Exists")
		} else if option == "-c" {
			newUser := User{UserName: username, Password: passwd}
			currentUsers = append(currentUsers, newUser)
		}
		fileJson.Users = currentUsers
	}

	data, err := json.Marshal(fileJson)
	if err != nil {
		fmt.Println("error converting json", err.Error(), fileJson)
	}
	writeToFile(data)
	fmt.Println("user created with password ", username, passwd)
}

func generateUser(domainName, domainType string) {
	var domName string
	var domType string
	if (domainName == "" || domainName == " ") && (domainType == "" || domainType == " ") {
		domName = "ionos-lft"
		domType = "com"
	} else if domainName == "" || domainName == " " {
		domName = "ionos-lft"
		domType = domainType
	} else if domainType == "" || domainType == " " {
		domName = domainName
		domType = "com"
	}

	email := createRandomEmailString(15, domName, domType)
	passwd := secureRandomAlphaNumericString(10)
	fmt.Println("generating user with email and password", email, passwd)
	updateUser(email, passwd, "-c")
}

func deleteUser(user string) {
	var fileJson UserJson
	if _, err := os.Stat(userConfigFileLoc); os.IsNotExist(err) {
		fmt.Println("No users to delete")
	} else {
		fileJson, _ = GetUsers()
		currentUsers := fileJson.Users
		for i := 0; i < len(currentUsers); i++ {
			currUser := currentUsers[i]
			if currUser.UserName == user {
				currentUsers = append(currentUsers[:i], currentUsers[i+1:]...)
				break
			}
		}
		fileJson.Users = currentUsers
	}

	data, err := json.Marshal(fileJson)
	if err != nil {
		fmt.Println("error converting json", err.Error(), fileJson)
	}
	writeToFile(data)
	fmt.Println("deleted user ", user)

}

func hashPassWord(passwd string) string {
	h := md5.New()
	h.Write([]byte(passwd))
	fmt.Println(hex.EncodeToString(h.Sum(nil)))
	//fmt.Printf("%x", h.Sum(nil))
	return hex.EncodeToString(h.Sum(nil))
}

func readFile(location string) []byte {
	dat, err := ioutil.ReadFile(location)
	if err != nil {
		fmt.Println("Error reading file", location, err.Error())
	}
	return dat
}

func createEmailString(username, domainName, domainType string) string {
	return username + "@" + domainName + "." + domainType
}

func createRandomEmailString(len int, domainName string, domainType string) string {
	//cpeIdBytes := readFile(cpeIdFileLoc)
	return (secureRandomAlphaString(len) + "@" + domainName + "." + domainType)
}
func secureRandomAlphaString(length int) string {
	return secureRandomString(alphaBytes, length)
}

func secureRandomAlphaNumericString(length int) string {
	return secureRandomString(alphaNumeric, length)
}

func secureRandomString(charset string, length int) string {
	result := make([]byte, length)
	bufferSize := int(float64(length) * 1.3)
	for i, j, randomBytes := 0, 0, []byte{}; i < length; j++ {
		if j%bufferSize == 0 {
			randomBytes = secureRandomBytes(bufferSize)
		}

		if idx := int(randomBytes[j%length] & letterIdxMask); idx < len(charset) {
			result[i] = charset[idx]
			i++
		}
	}
	return string(result)
}

func secureRandomBytes(length int) []byte {
	var randomBytes = make([]byte, length)
	_, err := rand.Read(randomBytes)
	if err != nil {
		fmt.Println("Unable to generate random bytes")
	}
	return randomBytes
}

func writeToFile(userData []byte) {
	_, err := os.OpenFile(userConfigFileLoc, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0600)

	if err != nil {
		fmt.Println("Error creating file", err.Error())
	}

	erro := ioutil.WriteFile(userConfigFileLoc, userData, 0600)
	if erro != nil {
		fmt.Println("Error writing to file", erro.Error())
	}
}

func GetUsers() (UserJson, error) {
	var userJsonObj UserJson
	fileData, err := ioutil.ReadFile(userConfigFileLoc)
	if err != nil {
		logr.Debug("Error reading user config file", err.Error())
		//return userJsonObj, err
	}
	err = json.Unmarshal(fileData, &userJsonObj)
	if err != nil {
		logr.Error("Error user json format", err.Error())
	}
	return userJsonObj, err
}
