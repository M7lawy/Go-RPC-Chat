package common

type Message struct {
	Sender    string
	Content   string
	IsNotify  bool 
}


type UserArgs struct {
	Username string
}

type UserReply struct {
	Success bool
	Message string
}


type SendMessageArgs struct {
	Username string
	Content  string
}

type SendMessageReply struct {
	Success bool
}


type GetMessagesArgs struct {
	Username string
}

type GetMessagesReply struct {
	Messages []Message
}


type ListUsersArgs struct{}

type ListUsersReply struct {
	Users []string
}