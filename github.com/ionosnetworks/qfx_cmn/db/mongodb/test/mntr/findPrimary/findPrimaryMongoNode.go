package main

import (
    "time"
    _"net"
    "fmt"
    _"crypto/tls"
    "gopkg.in/mgo.v2/bson"
    "gopkg.in/mgo.v2"
    "os"
)


const (
    DB_TYPE = "DB_TYPE"
    DB_SVC  = "DB_SVC"
    DB_PORT = "DB_PORT"
    Database = "ionos"
    Collection = "game"
)

func main() {
    
    if len(os.Args) != 2 {
       fmt.Fprintf(os.Stderr, "Usage %s <mongo-node-ip>\n", os.Args[0])
       os.Exit(1)
    }
    dbHost := os.Args[1]
    dbPort := "27017" 
    addrs := []string{dbHost + ":" + dbPort}
    session, err := mgo.DialWithInfo(&mgo.DialInfo{
                    Addrs: addrs,
                    Direct: false,
                    Timeout: 30 * time.Second,
                    })
    if err != nil {
        panic(err)
    }
    defer session.Close()
    fmt.Printf("Connected to replica set %v!\n", session.LiveServers())

    result := bson.M{}
    err = session.DB("admin").Run(bson.D{{"replSetGetStatus", 1}}, &result)
    fmt.Println()
    for key, value := range result {
        if key == "members" {
           v := value.([] interface {})
           for _, v1 := range v {
               m := v1.(bson.M)
               if m["stateStr"] == "PRIMARY" {
                  fmt.Println("PRIMARY   = ",m["name"])
               } else {
                  fmt.Println("SECONDARY = ",m["name"])
               }
           }
        }
    }
    fmt.Println("DONE")
}
