package main

import (
	"fmt"
	"net"
	"os/exec"
)

func main() {

	fmt.Println("Starting B")

	adr, _ := net.ResolveUDPAddr("udp", ":20011")

	conn, err := net.ListenUDP("udp", adr)

	if err != nil {
		fmt.Println(err)
		return
	} else {

	}

	err = exec.Command("gnome-terminal", "--", "go", "run", "/home/student/Gruppe54Sanntid/Sanntid-Heis1/Exercise4/A/A.go").Run()

	if err != nil {
		fmt.Println(err)
		fmt.Println("Could not run the A.go")
		return
	}

	msg := make([]byte, 1024)
	for {
		_, _ = conn.Read(msg)

		fmt.Println(string(msg))
	}

}
