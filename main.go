package main

import (
	"math/rand"
	"sort"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

var (
    redisClient *redis.Client
)

func main() {
    redisClient = redis.NewClient(&redis.Options{
        Addr:     "viaduct.proxy.rlwy.net:41040",
        Password: "lhfhjMiLA3dbjoDdeBLM5oefmOOpMfDo",
        DB:       0,
    })

    router := gin.Default()

    // Configure CORS middleware to allow all origins
    config := cors.DefaultConfig()
    config.AllowOrigins = []string{"*"}
    router.Use(cors.New(config))

    // Define routes
    router.GET("/", welcome)
    router.POST("/start-game", startGame)
    router.POST("/draw-card", drawCard)
    router.POST("/save-game", saveGame)
    router.GET("/leaderboard", getLeaderboard)

    router.Run(":10000")
}

// Handler for the root route
func welcome(c *gin.Context) {
    c.JSON(200, gin.H{"message": "Welcome to Exploding Kittens!"})
}

// Handler for starting a game
func startGame(c *gin.Context) {
    mainCards := []string{"KITTEN", "DIFFUSE", "SHUFFLE", "KITTEN", "EXPLODE"}

    shuffleDeck(mainCards)

    numCards := 5
    selectedCards := make([]string, numCards)
    rand.Seed(time.Now().UnixNano())
    for i := 0; i < numCards; i++ {
        randomIndex := rand.Intn(len(mainCards))
        selectedCards[i] = mainCards[randomIndex]
    }

    err := redisClient.Del(c, "deck").Err()
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }
    for _, card := range selectedCards {
        err := redisClient.LPush(c, "deck", card).Err()
        if err != nil {
            c.JSON(500, gin.H{"error": err.Error()})
            return
        }
    }

    err = redisClient.Set(c, "game_state", "started", 0).Err()
    if err != nil {
        c.JSON(500, gin.H{"error": err.Error()})
        return
    }

    c.JSON(200, gin.H{"message": "Game started successfully!", "deck": selectedCards})
}

// Handler for drawing a card
func drawCard(c *gin.Context) {
    gameState, err := redisClient.Get(c, "game_state").Result()
    if err != nil {
        c.JSON(500, gin.H{"error": "Game state not found"})
        return
    }
    if gameState != "started" {
        c.JSON(400, gin.H{"error": "Game not started yet"})
        return
    }

    card, err := redisClient.LPop(c, "deck").Result()
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to draw card"})
        return
    }

    switch card {
    case "KITTEN":
        c.JSON(200, gin.H{"message": "You drew a cat card 😼", "card": "KITTEN"})
    case "EXPLODE":
        c.JSON(200, gin.H{"message": "Game over! You drew an exploding kitten 💣", "card": "EXPLODE"})
    case "DIFFUSE":
        c.JSON(200, gin.H{"message": "You drew a defuse card 🙅‍♂️", "card": "DIFFUSE"})
    case "SHUFFLE":
        c.JSON(200, gin.H{"message": "You drew a shuffle card 🔀", "card": "SHUFFLE"})
    default:
        c.JSON(200, gin.H{"message": "Unknown card", "card": "UNKNOWN"})
    }
}

// Handler for saving a game
func saveGame(c *gin.Context) {
    gameState, err := redisClient.Get(c, "game_state").Result()
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to retrieve game state"})
        return
    }

    var user struct {
        Username string `json:"username"`
        Score    int    `json:"score"`
    }
    if err := c.ShouldBindJSON(&user); err != nil {
        c.JSON(400, gin.H{"error": "Invalid request body"})
        return
    }

    err = redisClient.Set(c, "saved_game_state", gameState, 0).Err()
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to save game state"})
        return
    }

    err = redisClient.Set(c, "user:"+user.Username, user.Score, 0).Err()
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to update leaderboard"})
        return
    }

    c.JSON(200, gin.H{"message": "Game state saved successfully"})
}

// Handler for retrieving the leaderboard
func getLeaderboard(c *gin.Context) {
    keys, err := redisClient.Keys(c, "user:*").Result()
    if err != nil {
        c.JSON(500, gin.H{"error": "Failed to retrieve leaderboard data"})
        return
    }

    type UserScore struct {
        Username string `json:"username"`
        Score    int    `json:"score"`
    }

    var leaderboard []UserScore
    for _, key := range keys {
        username := key[5:]
        score, err := redisClient.Get(c, key).Int()
        if err != nil {
            c.JSON(500, gin.H{"error": "Failed to retrieve user score"})
            return
        }
        leaderboard = append(leaderboard, UserScore{Username: username, Score: score})
    }

    sort.Slice(leaderboard, func(i, j int) bool {
        return leaderboard[i].Score > leaderboard[j].Score
    })

    c.JSON(200, leaderboard)
}

// Function to shuffle a deck of cards
func shuffleDeck(cards []string) {
    rand.Seed(time.Now().UnixNano())
    rand.Shuffle(len(cards), func(i, j int) {
        cards[i], cards[j] = cards[j], cards[i]
    })
}
