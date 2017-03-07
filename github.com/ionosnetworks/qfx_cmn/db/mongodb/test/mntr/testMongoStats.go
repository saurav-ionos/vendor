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

type Game struct {
    Winner       string    `bson:"winner"`
    OfficialGame bool      `bson:"official_game"`
    Location     string    `bson:"location"`
    StartTime    time.Time `bson:"start"`
    EndTime      time.Time `bson:"end"`
    Players      []Player  `bson:"players"`
}

type Player struct {
    Name   string    `bson:"name"`
    Decks  [2]string `bson:"decks"`
    Points uint8     `bson:"points"`
    Place  uint8     `bson:"place"`
}
func NewPlayer(name, firstDeck, secondDeck string, points, place uint8) Player {
    return Player{
        Name:   name,
        Decks:  [2]string{firstDeck, secondDeck},
        Points: points,
        Place:  place,
    }
}


func main() {
    dbHost := os.Getenv(DB_SVC)
    dbPort := os.Getenv(DB_PORT)
    if dbHost == "" {
       fmt.Println("DB Host IP not provided. Will now Panic")
       panic("No DB IP present")
       os.Exit(1)
    }
    if dbPort == "" {
       fmt.Println("DB Host Port not provided. Will now Panic")
       panic("No DB Port present")
       os.Exit(1)
    }
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
    err = session.DB("admin").Run(bson.D{{"serverStatus", 1}}, &result)
    fmt.Println("err=\n",err)
    fmt.Println(result)
    fmt.Println()
    fmt.Println()
    fmt.Println()
    fmt.Println()
    fmt.Println()
    time.Sleep(3 * time.Second)
    err = session.DB("ionos").Run("dbstats", &result)
    fmt.Println(result["dataSize"])
}
