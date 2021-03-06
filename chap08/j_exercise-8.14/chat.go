// Copyright © 2016 Alan A. A. Donovan & Brian W. Kernighan.
// License: https://creativecommons.org/licenses/by-nc-sa/4.0/

// See page 254.
//!+

// Chat is a server that lets clients chat with each other.
package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

//!+broadcaster
type client struct {
	Channel chan<- string // an outgoing message channel
	Name    string
}

const (
	timeout = 5 * time.Minute
)

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string) // all incoming client messages
)

func broadcaster() {
	clients := make(map[client]bool) // all connected clients
	for {
		select {
		case msg := <-messages:
			// Broadcast incoming message to all
			// clients' outgoing message channels.
			for cli := range clients {
				cli.Channel <- msg
			}

		case cli := <-entering:
			clients[cli] = true
			cli.Channel <- "welcome from the users:"
			for user := range clients {
				cli.Channel <- user.Name
			}

		case cli := <-leaving:
			delete(clients, cli)
			close(cli.Channel)
		}
	}
}

//!-broadcaster

//!+handleConn
func handleConn(conn net.Conn) {
	ch := make(chan string) // outgoing client messages
	who := askName(conn)
	go clientWriter(conn, ch)
	ch <- "You are " + who
	messages <- who + " has arrived"
	entering <- client{ch, who}
	// read messages from connection in goroutine
	cli := make(chan string)
	go func() {
		input := bufio.NewScanner(conn)
		for input.Scan() {
			cli <- who + ": " + input.Text()
		}
	}()
	// start timeout via channel
	timeoutChannel := time.After(timeout)
loop:
	for {
		select {
		case msg := <-cli:
			messages <- msg
			timeoutChannel = time.After(timeout)
		case <-timeoutChannel:
			break loop
		}
	}

	leaving <- client{ch, who}
	messages <- who + " has left"
	conn.Close()
}

func askName(conn net.Conn) string {
	fmt.Fprintln(conn, "please type your name:")
	name, _ := bufio.NewReader(conn).ReadString('\n')
	return name[:len(name)-1] //last byte removed: \n
}

// func scanClient(ctx context.Context, conn net.Conn, out chan<- string) {
// 	input := bufio.NewScanner(conn)
// 	for input.Scan() {
// 		out <- who + ": " + input.Text()
// 	}
// }

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg) // NOTE: ignoring network errors
	}
}

//!-handleConn

//!+main
func main() {
	listener, err := net.Listen("tcp", "localhost:8000")
	if err != nil {
		log.Fatal(err)
	}

	go broadcaster()
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Print(err)
			continue
		}
		go handleConn(conn)
	}
}

//!-main
