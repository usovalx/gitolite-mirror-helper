package main

import (
    "fmt"
    "log"
    "os"
)

func main() {
    fmt.Printf("Hello\n")
    procmon(
        log.New(os.Stderr, "", log.LstdFlags),
        "echo",
        []string{"echo", "foo", "bar"})
}
