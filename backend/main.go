// main.go
package main

import (
	"errors"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"messaging-app-backend/model"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	// Active websocket connections
	clients = make(map[uint]*model.WSConnection)
	// Broadcast channel for messages
	broadcast = make(chan model.Message)
	// Websocket upgrader
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true // In production, check origin
		},
	}
	// Database connection
	db *gorm.DB
)

// initDB initializes the database connection
func initDB() {
	var err error
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"))
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto migrate the schemas
	err = db.AutoMigrate(&model.User{}, &model.Message{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}
}

// handleConnections handles websocket connections
func handleConnections(c *gin.Context) {
	// Get token from query parameter
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
		return
	}

	// Parse and validate token
	claims := jwt.MapClaims{}
	parsedToken, err := jwt.ParseWithClaims(token, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})

	if err != nil || !parsedToken.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	userID := uint(claims["user_id"].(float64))
	log.Printf("Current user Id in 'handleConnections'='%v'", userID)

	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to set websocket upgrade: %+v", err)
		return
	}
	defer ws.Close()

	// Register new client
	clients[userID] = &model.WSConnection{
		UserID: userID,
		Conn:   ws,
	}

	for {
		var msg model.Message
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Error reading message: %v", err)
			delete(clients, userID)
			break
		}

		msg.SenderID = userID
		db.Create(&msg)
		broadcast <- msg
	}
}

// handleMessages broadcasts messages to intended recipients
func handleMessages() {
	for {
		msg := <-broadcast

		// Find target user id from conversation entity in the db
		// Get second user id from conversation's participants
		// Find all active participants in the conversation except the sender
		var participants []model.ConversationParticipant
		if err := db.Where(
			"conversation_id = ? AND user_id != ? AND left_at IS NULL",
			msg.ConversationID,
			msg.SenderID,
		).Find(&participants).Error; err != nil {
			log.Printf("Error finding conversation participants: %v", err)
			continue
		}

		// Send message to all participants who are online
		for _, participant := range participants {
			if client, ok := clients[participant.UserID]; ok {
				err := client.Conn.WriteJSON(msg)
				if err != nil {
					log.Printf("Error sending message to user %s: %v", participant.UserID, err)
					client.Conn.Close()
					delete(clients, participant.UserID)
				}
			}
		}
	}
}

// registerUser handles user registration
func registerUser(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}
	user.Password = string(hashedPassword)

	// Save user
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	log.Printf("Created user: %v", user)
	c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

// loginUser handles user login
func loginUser(c *gin.Context) {
	var loginData struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user model.User
	if err := db.Where("username = ?", loginData.Username).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginData.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
	})

	log.Printf("Current user Id in 'loginUser'='%v'", user.ID)

	tokenString, err := token.SignedString([]byte(os.Getenv("JWT_SECRET"))) // Use environment variable in production
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

// authMiddleware validates JWT tokens
func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing authorization header"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("JWT_SECRET")), nil // Use environment variable in production
		})

		if err != nil || !token.Valid {
			log.Printf("Unable to validate token: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		claims := token.Claims.(jwt.MapClaims)
		c.Set("user_id", uint(claims["user_id"].(float64)))
		c.Next()
	}
}

// validateToken checks if the token is valid and returns user info
func validateToken(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user model.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Don't send password in response
	user.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}

// getUsers returns list of all users except the requesting user
func getUsers(c *gin.Context) {
	currentUserID := c.GetUint("user_id")

	var users []model.User
	if err := db.Where("id != ?", currentUserID).Find(&users).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Remove passwords from response
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// getConversation fetches messages from conversation between users
func getConversation(context *gin.Context) {
	conversationId, err := strconv.ParseUint(context.Param("id"), 10, 64)
	if err != nil {
		log.Printf("Error parsing conversation id: %v", err)
		context.JSON(http.StatusBadRequest, gin.H{"error": "Invalid conversation id"})
		return
	}

	var conversation model.Conversation

	if err := db.
		Where("id = ?", conversationId).
		Find(&conversation).Error; err != nil {

		log.Printf("Failed to fetch conversation: %v", err)
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch conversation"})
		return
	}

	context.JSON(http.StatusOK, gin.H{"conversation": conversation})
}

// FindExistingConversation Searches the conversations in the db that matches userId1 and userId2 specified
// in the parameters
// returns error if no conversations with specified parameters are found
func FindExistingConversation(userId1, userId2 uint) (*model.Conversation, error) {
	var convIds []uint
	if err := db.
		Table("conversation_participants").
		Where("user_id = ? AND left_at IS NULL", userId1).
		Pluck("conversation_id", &convIds).Error; err != nil {
		return nil, fmt.Errorf("error finding conversations for user1: %v", err)
	}

	// If no conversations found - return nil
	if len(convIds) == 0 {
		return nil, nil
	}

	// Among conversations found - search for conv where userId2 is also a participant
	var conversation model.Conversation
	err := db.
		Joins("JOIN conversation_participants cp1 ON cp1.conversation_id = conversations.id").
		Joins("JOIN conversation_participants cp2 ON cp2.conversation_id = conversations.id").
		Where("cp1.user_id = ? AND cp2.user_id = ? AND cp1.left_at IS NULL AND cp2.left_at IS NULL", userId1, userId2).
		Where("conversations.deleted_at IS NULL").
		Preload("Participants").
		Preload("Messages").
		First(&conversation).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("error finding conversation: %v", err)
	}

	return &conversation, nil
}

func CreateOrGetConversation(c *gin.Context) {
	// Get userIds from request params
	userId1, err := strconv.ParseUint(c.Param("userId1"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user1 id"})
		return
	}

	userId2, err := strconv.ParseUint(c.Param("userId2"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user2 id"})
		return
	}

	// Try to find an existing conversation
	conversation, err := FindExistingConversation(uint(userId1), uint(userId2))
	if err != nil {
		// Error while working with the DB
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Conversation found in the DB
	if conversation != nil {
		c.JSON(http.StatusOK, gin.H{"conversation": conversation})
		return
	}

	// Conversation not found - create a new one
	newConv := &model.Conversation{}

	// Initiate a transaction
	tx := db.Begin()

	if err := tx.Create(&newConv).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation"})
		return
	}

	// Create conversation participants
	participants := []model.ConversationParticipant{
		{
			ConversationID: newConv.ID,
			UserID:         uint(userId1),
			JoinedAt:       time.Now(),
		},
		{
			ConversationID: newConv.ID,
			UserID:         uint(userId2),
			JoinedAt:       time.Now(),
		},
	}

	if err := tx.Create(&participants).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create conversation participants"})
		return
	}

	// Go through with the transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Load participants
	if err := db.Preload("Participants").First(newConv, newConv.ID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load participants"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"conversation": newConv})
}

func main() {
	initDB()

	router := gin.Default()

	// Add CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Public routes
	router.POST("/register", registerUser)
	router.POST("/login", loginUser)

	// TODO: Move to protected
	router.GET("/conversation/:userId1/:userId2", CreateOrGetConversation)

	// Protected routes
	router.GET("/ws", handleConnections)

	auth := router.Group("/")
	auth.Use(authMiddleware())
	{
		auth.GET("/validate-token", validateToken)
		auth.GET("/users", getUsers)
	}

	// Start message handling goroutine
	go handleMessages()

	log.Fatal(router.Run(":8080"))
}
