package model

import (
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
	"time"
)

type Conversation struct {
	gorm.Model
	Messages     []Message                 `gorm:"foreignKey:ConversationID;references:ID" json:"messages"`
	Participants []ConversationParticipant `gorm:"foreignKey:ConversationID;references:ID" json:"participants"`
}

// Message represents a chat message
type Message struct {
	gorm.Model
	Content        string `json:"content"`
	ConversationID uint   `json:"conversation_id"`
	SenderID       uint   `json:"sender_id"`
}

// User represents the user model
type User struct {
	gorm.Model
	Username string `json:"username" gorm:"unique"`
	Password string `json:"password"`
}

// WSConnection maintains the websocket connection info
type WSConnection struct {
	UserID uint
	Conn   *websocket.Conn
}

type ConversationParticipant struct {
	ConversationID uint
	UserID         uint
	JoinedAt       time.Time
	LeftAt         *time.Time
}
