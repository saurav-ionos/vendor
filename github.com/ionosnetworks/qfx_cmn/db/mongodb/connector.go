package mongodb
import (
"time"
"gopkg.in/mgo.v2"
"strings"
"github.com/pkg/errors"
)


func GetClusterSession(addrs []string, port string) (*mgo.Session, error) {
     if len(addrs) <= 0 {
        return nil, errors.New("No MongoDB IP provided")
     }
     port = strings.TrimSpace(port)
     if len(port) <= 0 {
        return nil, errors.New("No MongoDB Port provided")
     }
     var addrsInfo []string
     for _, addr := range addrs {
         addr = strings.TrimSpace(addr)
         addrsInfo = append(addrsInfo, addr + ":" + port)
     }
     if len(addrsInfo) <= 0 {
        return nil, errors.New("No MongoDB IP provided")
     }
     session, err := mgo.DialWithInfo(&mgo.DialInfo{
                    Addrs: addrsInfo,
                    Direct: false,
                    Timeout: 30 * time.Second,
                    })
     return session, err
}

func GetDirectSession(addr string, port string) (*mgo.Session, error) {
     addr = strings.TrimSpace(addr)
     if len(addr) <= 0 {
        return nil, errors.New("No MongoDB IP provided")
     }
     port = strings.TrimSpace(port)
     if len(port) <= 0 {
        return nil, errors.New("No MongoDB Port provided")
     }
     var addrsInfo []string
     addrsInfo = append(addrsInfo, addr + ":" + port)
     if len(addrsInfo) <= 0 {
        return nil, errors.New("No MongoDB IP provided")
     }
     session, err := mgo.DialWithInfo(&mgo.DialInfo{
                    Addrs: addrsInfo,
                    Direct: true,
                    Timeout: 30 * time.Second,
                    })
     return session, err
}
