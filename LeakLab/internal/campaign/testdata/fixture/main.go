// Command fixture simule les fins de processus que la campagne doit classer (TestUC001_BR4).
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	switch os.Args[1] {
	case "pass":
		fmt.Println("PASS")
	case "fail":
		fmt.Println("    driver_test.go:38: LEAKLAB-WITNESS : assertion fausse")
		os.Exit(1)
	case "hang":
		time.Sleep(time.Hour)
	case "deadlock":
		ch := make(chan int)
		ch <- 1
	case "leak":
		fmt.Println("LEAKLAB-LEAK goroutines=10")
	}
}
