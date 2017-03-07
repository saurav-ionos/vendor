package main

import (
    "time"
    _"net"
    "fmt"
    _"crypto/tls"
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

    game := Game{
        Winner:       "Dave",
        OfficialGame: true,
        Location:     "Austin",
        StartTime:    time.Date(2015, time.February, 12, 04, 11, 0, 0, time.UTC),
        EndTime:      time.Date(2015, time.February, 12, 05, 54, 0, 0, time.UTC),
        Players: []Player{
            NewPlayer("Dave", "Wizards", "Steampunk", 21, 1),
            NewPlayer("Javier", "Zombies", "Ghosts", 18, 2),
            NewPlayer("George", "Aliens", "Dinosaurs", 17, 3),
            NewPlayer("Seth", "Spies", "Leprechauns", 10, 4),
        },
    }

    coll := session.DB(Database).C(Collection)
    if err := coll.Insert(game); err != nil {
        panic(err)
    }
    time.Sleep(3 * time.Second)
}
