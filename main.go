package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("Sleeping 5 seconds...")
	time.Sleep(5 * time.Second)

	output, err := executeCommand(9797, "inspectheap", "")

	if err != nil {
		fmt.Printf("Error while executing command.: %v", err)
	} else {
		fmt.Println(string(output))
	}
}