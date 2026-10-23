package main

import (
	"fmt"
	"net"
	"net/rpc"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"go-rpc-chat/common"
)

type ChatServer struct {
	mu       sync.Mutex
	clients  map[string]chan common.Message // Per-client channel acting as a buffer/queue
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		clients: make(map[string]chan common.Message),
	}
}


func (s *ChatServer) broadcast(msg common.Message, excludeSender string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for username, ch := range s.clients {
		if username != excludeSender {
			select {
			case ch <- msg:
			default:
				
			}
		}
	}
}


func (s *ChatServer) Join(args common.UserArgs, reply *common.UserReply) error {
	s.mu.Lock()
	if args.Username == "" {
		reply.Success = false
		reply.Message = "Username cannot be empty."
		s.mu.Unlock()
		return nil
	}

	if _, exists := s.clients[args.Username]; exists {
		reply.Success = false
		reply.Message = fmt.Sprintf("Username '%s' is already taken.", args.Username)
		s.mu.Unlock()
		return nil
	}

	
	s.clients[args.Username] = make(chan common.Message, 100)
	s.mu.Unlock()

	reply.Success = true
	reply.Message = "Successfully joined the chat."

	
	s.broadcast(common.Message{
		Sender:   "SYSTEM",
		Content:  fmt.Sprintf("*** %s joined the chat ***", args.Username),
		IsNotify: true,
	}, args.Username)

	fmt.Printf("[SERVER LOG] User '%s' joined.\n", args.Username)
	return nil
}


func (s *ChatServer) Leave(args common.UserArgs, reply *common.UserReply) error {
	s.mu.Lock()
	ch, exists := s.clients[args.Username]
	if !exists {
		s.mu.Unlock()
		reply.Success = false
		reply.Message = "User not found."
		return nil
	}

	delete(s.clients, args.Username)
	close(ch)
	s.mu.Unlock()

	reply.Success = true
	reply.Message = "Successfully left the chat."

	// Notify other clients
	s.broadcast(common.Message{
		Sender:   "SYSTEM",
		Content:  fmt.Sprintf("*** %s left the chat ***", args.Username),
		IsNotify: true,
	}, args.Username)

	fmt.Printf("[SERVER LOG] User '%s' left.\n", args.Username)
	return nil
}


func (s *ChatServer) SendMessage(args common.SendMessageArgs, reply *common.SendMessageReply) error {
	s.mu.Lock()
	_, exists := s.clients[args.Username]
	s.mu.Unlock()

	if !exists {
		reply.Success = false
		return fmt.Errorf("user '%s' is not registered", args.Username)
	}

	msg := common.Message{
		Sender:   args.Username,
		Content:  args.Content,
		IsNotify: false,
	}

	s.broadcast(msg, args.Username)
	reply.Success = true
	return nil
}


func (s *ChatServer) GetMessages(args common.GetMessagesArgs, reply *common.GetMessagesReply) error {
	s.mu.Lock()
	ch, exists := s.clients[args.Username]
	s.mu.Unlock()

	if !exists {
		return nil 
	}

	var messages []common.Message
	
Loop:
	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				break Loop
			}
			messages = append(messages, msg)
		default:
			break Loop
		}
	}

	reply.Messages = messages
	return nil
}


func (s *ChatServer) ListUsers(args common.ListUsersArgs, reply *common.ListUsersReply) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	users := make([]string, 0, len(s.clients))
	for user := range s.clients {
		users = append(users, user)
	}
	reply.Users = users
	return nil
}

func main() {
	chatServer := NewChatServer()
	err := rpc.Register(chatServer)
	if err != nil {
		fmt.Printf("Error registering RPC server: %v\n", err)
		return
	}

	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Printf("Server listen error: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Println("Server running on port 8080...")

	// Handle graceful shutdown (Ctrl+C)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down server gracefully...")
		listener.Close()
		os.Exit(0)
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			select {
			case <-sigChan:
				return
			default:
				continue
			}
		}
		go rpc.ServeConn(conn)
	}
}