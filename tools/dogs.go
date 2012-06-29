package main

import (
	"femebe"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	_ "github.com/bmizerany/pq"
	"database/sql"
)

// Automatically chooses between unix sockets and tcp sockets for
// listening
func autoListen(place string) (net.Listener, error) {
	if strings.Contains(place, "/") {
		return net.Listen("unix", place)
	}

	return net.Listen("tcp", place)
}

// Automatically chooses between unix sockets and tcp sockets for
// dialing.
func autoDial(place string) (net.Conn, error) {
	if strings.Contains(place, "/") {
		return net.Dial("unix", place)
	}

	return net.Dial("tcp", place)
}

type handlerFunc func(
	client femebe.MessageStream,
	server femebe.MessageStream,
	errch chan error)

type proxyBehavior struct {
	toFrontend handlerFunc
	toServer   handlerFunc
}

func (pbh *proxyBehavior) start(
	client femebe.MessageStream,
	server femebe.MessageStream) (errch chan error) {

	errch = make(chan error)

	go pbh.toFrontend(client, server, errch)
	go pbh.toServer(client, server, errch)
	return errch
}

var dog = proxyBehavior{
	toFrontend: func(client femebe.MessageStream,
		server femebe.MessageStream, errch chan error) {
		for {
			msg, err := server.Next()
			if err != nil {
				errch <- err
				return
			}

			err = client.Send(msg)
			if err != nil {
				errch <- err
				return
			}
		}
	},
	toServer: func(client femebe.MessageStream,
		server femebe.MessageStream, errch chan error) {
		for {
			msg, err := client.Next()
			if err != nil {
				errch <- err
				return
			}

			err = server.Send(msg)
			if err != nil {
				errch <- err
				return
			}
		}
	},
}

func lookupServer(database string) (serverAddr string, err error) {
	db, _ := sql.Open("postgres", "")

	fmt.Printf("DBNAME: %v", database)
	r, err := db.Query("SELECT serveraddr FROM databases WHERE dbname=$1", database)
	if err != nil {
		fmt.Printf("No query: %v", err)
		return "", err
	}

	defer r.Close()

	if !r.Next() {
		fmt.Printf("No next: %v", err)
		return "", nil
	}

	var v string
	
	err = r.Scan(&v)
	if (err != nil) {
		fmt.Printf("No scan")
		return "", nil
	}
	
	return v, nil
}

func findServer(client femebe.MessageStream) (server femebe.MessageStream, err error){
	msg, err := client.Next()
	if err != nil {
		return nil, err
	}

	var dbname string

	if femebe.IsStartupMessage(msg) {
		startupMsg := femebe.ReadStartupMessage(msg)
		fmt.Println("Got startup message:")
		for key, value := range startupMsg.Params {
			fmt.Printf("\t%v: %v\n", key, value)			
		}
		dbname = startupMsg.Params["database"]
		fmt.Printf("dbname: %v\n", dbname) 
	}

	serverAddr, _ := lookupServer(dbname)

	fmt.Printf("serverAddr: %v\n", serverAddr)

	serverConn, err := autoDial(serverAddr)
	if err != nil {
		fmt.Printf("Could not connect to server: %v\n", err)
	}

	server = femebe.NewMessageStream("Server", serverConn, serverConn)
	server.Send(msg)

	return server, nil
}

// Generic connection handler
//
// This redelegates to more specific proxy handlers that contain the
// main proxy loop logic.
func handleConnection(proxy proxyBehavior, clientConn net.Conn) {
	var err error

	// Log disconnections
	defer func() {
		if err != nil && err != io.EOF {
			fmt.Printf("Session exits with error: %v\n", err)
		} else {
			fmt.Printf("Session exits cleanly\n")
		}
	}()

	defer clientConn.Close()

	c := femebe.NewMessageStream("Client", clientConn, clientConn)

	s, err := findServer(c)
	if (err != nil) { return }

	done := proxy.start(c, s)
	err = <-done
}

// Startup and main client acceptance loop
func main() {
	if len(os.Args) != 3 {
		fmt.Printf("Usage: dog\n")
		os.Exit(1)
	}

	ln, err := autoListen(os.Args[1])
	if err != nil {
		fmt.Printf("Could not listen on address: %v\n", err)
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		go handleConnection(dog, conn)
	}

	fmt.Println("simpleproxy quits successfully")
	return
}
