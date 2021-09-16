package main

import "github.com/mgutz/ansi"

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
        "time"
)

// ##### 1

type client struct {
	Conn net.Conn
	Name string
        Room string
        JoinedAt string
	TextColour string
}

type message struct {
	MessageText string
        Room string
	User string
	TextColour string
}

var clients = make(map[string]*client)
var messages = make(chan message)


func main() {

	msg := ansi.Color("started", "red")
        log.Println(fmt.Sprintf(msg))

	listener, err := net.Listen("tcp", "0.0.0.0:1234")
	if err != nil {
		log.Fatalln(err)
	}
	defer listener.Close()
	go broadcastMsg()
	for {
		con, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go handleConn(con)
	}

}

func handleConn(con net.Conn) {

	helpMsg:="\n\n/shout <msg>\n"
	helpMsg+="/whisper <user> <msg>\n"
	helpMsg+="/name <newname>\n"
	helpMsg+="/go <room>\n"
	helpMsg+="/join <user>		-go to the same room as <user>\n"
	helpMsg+="/who			-lists all connected users\n"
	helpMsg+="/look			-lists users in the same room\n"
	helpMsg+="/colour <colour>	-change the colour that is used when you talk/shout/whisper\n"
	helpMsg+="/help\n"
	helpMsg+="/quit\n\n"
	helpMsg = ansi.Color(helpMsg, "green")

	defer con.Close()
	clientReader := bufio.NewReader(con)
	addr := con.RemoteAddr().String()
        currentTime := time.Now().String()
	client := &client{
		Conn: con,
		Name: "User" + fmt.Sprintf("%d", len(clients)+1),
                Room: "None",
                JoinedAt: currentTime,
                TextColour: "white",
	}
	clients[addr] = client
	// ##### 1
	defer handleDisconn(addr)

	welcomeMsg := "Welcome " + client.Name + "! /help for help\n\n"
	welcomeMsg = ansi.Color(welcomeMsg, "yellow")
	if _, err := client.Conn.Write([]byte(welcomeMsg)); err != nil {
		log.Printf("failed to send to client: %v\n", err)
	}


	for {
		clientMsg, err := clientReader.ReadString('\n')
		if err == nil {
			clientMsg := strings.TrimSpace(clientMsg)

			if strings.HasPrefix(clientMsg, "/") {
				if strings.HasPrefix(clientMsg, "/help") {
                                	if _, err := client.Conn.Write([]byte(helpMsg)); err != nil {
                                 		log.Printf("failed to send to client: %v\n", err)
                                 	}
				}
				if strings.HasPrefix(clientMsg, "/colour ") {
					colour := strings.SplitAfterN(clientMsg, "/colour ", 2)[1]
					client.TextColour = colour
					clients[addr] = client
				}
				if strings.HasPrefix(clientMsg, "/shout ") {
					shout := strings.SplitAfterN(clientMsg, "/shout ", 2)[1]
                                        messages <- message{ MessageText: client.Name + " shouted: " + shout, Room: "", User: "", TextColour: client.TextColour}
				}
				if strings.HasPrefix(clientMsg, "/whisper ") {
                                        substr := clientMsg[9:]
					targetUser := substr[0:strings.IndexAny(substr, " ")]
					whisper := substr[strings.IndexAny(substr, " "):]
                                        messages <- message{ MessageText: client.Name + " whispered: " + whisper, Room: "", User: targetUser, TextColour: client.TextColour}
				}
				if clientMsg == "/quit" {
					if _, err = con.Write([]byte("Than you for joining our chat!\n")); err != nil {
						log.Printf("failed to respond to client: %v\n", err)
					}
					log.Println("client requested server to close the connection so closing")
					return
				}
 				if strings.HasPrefix(clientMsg, "/name ") {
					newName := strings.SplitAfterN(clientMsg, "/name ", 2)[1]
					messages <- message{ MessageText: client.Name + " has changed name to: " + newName, Room: "", User: "", }
					client.Name = newName
					clients[addr] = client
				}
				if strings.HasPrefix(clientMsg, "/go ") {
					oldRoom := client.Room
					newRoom := strings.SplitAfterN(clientMsg, "/go ", 2)[1]
                                        messages <- message{ MessageText: client.Name + " has left room: " + oldRoom, Room: oldRoom, User: ""}
					client.Room = newRoom
					clients[addr] = client
                                        messages <- message{ MessageText: client.Name + " has entered room: " + newRoom, Room: newRoom, User: "" }
				}
 				if strings.HasPrefix(clientMsg, "/join ") {
					joinUser := strings.SplitAfterN(clientMsg, "/join ", 2)[1]
					for _, other_client := range clients {
                                                if other_client.Name == joinUser {
							client.Room = other_client.Room
		                                        messages <- message{ MessageText: client.Name + " has joined: '" + joinUser + "' in Room '" + other_client.Room+ "'", Room: "", User: "" }
							continue
						}
					}
					clients[addr] = client
				}

				if strings.HasPrefix(clientMsg, "/room") {
                                	if _, err := client.Conn.Write([]byte("You are in room: " + client.Room + "!\n")); err != nil {
                                 		log.Printf("failed to send to client: %v\n", err)
                                 	}
				}

				if strings.HasPrefix(clientMsg, "/time") {
                                        currentTime := time.Now().String()
                                	if _, err := client.Conn.Write([]byte("You joined the chat at: " + client.JoinedAt + "!\nThe time is now: " + currentTime + "\n")); err != nil {
                                 		log.Printf("failed to send to client: %v\n", err)
                                 	}
				}

				if strings.HasPrefix(clientMsg, "/who") {
					for other_ipaddr, other_client := range clients {
						if _, err := client.Conn.Write([]byte(other_client.Name + " from " + other_ipaddr + " is in room '" + other_client.Room + "' and joined at " + other_client.JoinedAt + "\n")); err != nil {
							log.Printf("failed to send to client: %v\n", err)
						}
					}
				}

				if strings.HasPrefix(clientMsg, "/look") {
					for other_ipaddr, other_client := range clients {
                                                if other_client.Room == client.Room {
							if _, err := client.Conn.Write([]byte(other_client.Name + " from " + other_ipaddr + " is in room '" + other_client.Room + "' and joined at " + other_client.JoinedAt + "\n")); err != nil {
								log.Printf("failed to send to client: %v\n", err)
							}
						}
					}
				}

			} else {
				log.Printf("Received: %s, bytes: %d \n", string(clientMsg), len(clientMsg))
                                 messages <- message{ MessageText: fmt.Sprintf("(%s) %s", client.Name, string(clientMsg)), Room: client.Room, User: "", TextColour: client.TextColour}
			}
		}
	}

}

func broadcastMsg() {

	for {
		msg := <-messages
                MessageText := msg.MessageText
                targetRoom := msg.Room
                targetUser := msg.User
                messageColour := msg.TextColour
		if messageColour != "" {
			MessageText = ansi.Color(MessageText, messageColour)
		}
		log.Println("received message from channel: " + MessageText + " for room: " + targetRoom + " and user " + targetUser + "(" + messageColour + ")")
		// ##### value is not con - its a client
		for ipaddr, client := range clients {
			if targetRoom == "" || targetRoom == client.Room {
				if targetUser == "" || targetUser == client.Name {
					log.Printf("broadcasting MessageText to ip: %v\n", ipaddr)
					if _, err := client.Conn.Write([]byte(ipaddr + " : " + MessageText + "\n")); err != nil {
						log.Printf("failed to send to client: %v\n", err)
					}
				}
			}
		}
	}

}

// ###### 2 handling other_ disconnection

func handleDisconn(addr string) {
	client := clients[addr]
        messages <- message{ MessageText: fmt.Sprintf("user '%s' has left the chat", client.Name), Room: "", User: "",}
	delete(clients, addr)

}
