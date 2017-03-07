package utils
import  "github.com/pborman/uuid"

func GetGuidStr() string { //can return nil
     uuid := uuid.NewRandom()
     if uuid == nil {
        return ""
     }
     return uuid.String()
}
