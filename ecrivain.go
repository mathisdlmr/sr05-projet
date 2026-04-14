package main

import (
    "fmt"
    "time"
	"strconv"
)

var fieldsep = "/"
var keyvalsep = "="

func msg_format(key string, val string) string {
    return fieldsep + keyvalsep + key + keyvalsep + val
}

func main() {
    var message = "coucou"

    for {
		fmt.Println(msg_format("msg", message))
        time.Sleep(time.Duration(2) * time.Second)
    }
}