package main

import (
	"bufio"
	"fmt"
	"net/rpc"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"go-rpc-chat/common"
)

var (
	client   *rpc.Client
	username string
	mu       sync.Mutex
	running  bool = true
)


func pollMessages() {
	for {
		mu.Lock()
		if !running {
			mu.Unlock()
			return
		}
		mu.Unlock()

		args := common.GetMessagesArgs{Username: username}
		var reply common.GetMessagesReply

		err := client.Call("ChatServer.GetMessages", args, &reply)
		if err == nil && len(reply.Messages) > 0 {
			for _, msg := range reply.Messages {
				if msg.IsNotify {
					fmt.Printf("\n[NOTIFICATION] %s\n> ", msg.Content)
				} else {
					fmt.Printf("\n[%s]: %s\n> ", msg.Sender, msg.Content)
				}
			}
		}

		time.Sleep(500 * time.Millisecond) // Poll every 500ms
	}
}

func leaveChat() {
	mu.Lock()
	if !running {
		mu.Unlock()
		return
	}
	running = false
	mu.Unlock()

	args := common.UserArgs{Username: username}
	var reply common.UserReply
	_ = client.Call("ChatServer.Leave", args, &reply)
	fmt.Println("\nDisconnected from chat.")
}

func main() {
	var err error
	client, err = rpc.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Printf("Could not connect to server: %v\n", err)
		return
	}
	defer client.Close()

	reader := bufio.NewReader(os.Stdin)

	
	for {
		fmt.Print("Enter unique username: ")
		input, _ := reader.ReadString('\n')
		username = strings.TrimSpace(input)

		if username == "" {
			continue
		}

		args := common.UserArgs{Username: username}
		var reply common.UserReply
		err := client.Call("ChatServer.Join", args, &reply)
		if err != nil {
			fmt.Printf("RPC Error: %v\n", err)
			return
		}

		if reply.Success {
			fmt.Println(reply.Message)
			break
		} else {
			fmt.Printf("Join failed: %s\n", reply.Message)
		}
	}

	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		leaveChat()
		os.Exit(0)
	}()

	
	go pollMessages()

	fmt.Println("\n--- CHAT STARTED ---")
	fmt.Println("Commands:")
	fmt.Println("  /users  - List all connected users")
	fmt.Println("  /exit   - Disconnect and leave")
	fmt.Println("  Type any message and press Enter to send.")
	fmt.Println("--------------------")


	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}

		text := strings.TrimSpace(line)
		if text == "" {
			continue
		}

		switch text {
		case "/exit":
			leaveChat()
			return
		case "/users":
			var reply common.ListUsersReply
			err := client.Call("ChatServer.ListUsers", common.ListUsersArgs{}, &reply)
			if err == nil {
				fmt.Printf("Connected users: %s\n", strings.Join(reply.Users, ", "))
			} else {
				fmt.Printf("Error fetching user list: %v\n", err)
			}
		default:
			// Send message
			args := common.SendMessageArgs{
				Username: username,
				Content:  text,
			}
			var reply common.SendMessageReply
			err := client.Call("ChatServer.SendMessage", args, &reply)
			if err != nil {
				fmt.Printf("Error sending message: %v\n", err)
			}
		}
	}
}