package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
)

// DROP TABLE IF EXISTS todo;
// CREATE TABLE todo (
//   id         INT AUTO_INCREMENT NOT NULL,
//   item       VARCHAR(255) NOT NULL,
//   completed  tinyint(1) unsigned DEFAULT '0',
//   PRIMARY KEY (`id`)
// );

// INSERT INTO todo
//   (item, completed)
// VALUES
//   ('Create todo table', 1),
//   ('Tell about it', 0);

type Todo struct {
	ID        int    `json:"id,omitempty"`
	Item      string `json:"item"`
	Completed uint8  `json:"completed,omitempty"`
}

var db *sql.DB

func getTodoById(id string) (Todo, error) {
	var todo Todo

	row := db.QueryRow("SELECT * FROM todo WHERE id = ?", id)
	if err := row.Scan(&todo.ID, &todo.Item, &todo.Completed); err != nil {
		if err == sql.ErrNoRows {
			return todo, fmt.Errorf("todoById %s: no such todo", id)
		}
		return todo, fmt.Errorf("todoById %s: %v", id, err)
	}
	return todo, nil
}

func getTodo(context *gin.Context) {
	id := context.Param("id")
	todo, err := getTodoById(id)
	if err != nil {
		context.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	context.IndentedJSON(http.StatusOK, &todo)
}

func getTodos(c *gin.Context) {
	var todos []Todo

	rows, err := db.Query("SELECT * FROM todo")
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.ID, &todo.Item, &todo.Completed); err != nil {
			c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		todos = append(todos, todo)
	}
	if err := rows.Err(); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, &todos)
}

func createTodo(c *gin.Context) {
	var todo Todo
	if err := c.BindJSON(&todo); err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
		return
	}

	if todo.Item == "" {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": "Item is required"})
		return
	}

	_, err := db.Exec("INSERT INTO todo (item, completed) VALUES (?, ?)", todo.Item, todo.Completed)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusCreated, todo)
}

func toggleTodoStatus(c *gin.Context) {
	id := c.Param("id")

	_, err := db.Exec(`
	UPDATE todo
	SET completed = CASE WHEN Completed = 0 THEN 1 ELSE 0 END
	WHERE id = ?;
`, id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	todo, err := getTodoById(id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, todo)
}

func updateTodo(c *gin.Context) {
	id := c.Param("id")

	var todoData Todo
	if err := c.ShouldBindJSON(&todoData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid data"})
		return
	}
	_, err := db.Exec(`
	UPDATE todo
	SET item = ?, completed = ?
	WHERE id = ?;
`, todoData.Item, todoData.Completed, id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	todo, err := getTodoById(id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, todo)
}

func deleteTodo(c *gin.Context) {
	id := c.Param("id")
	todo, err := getTodoById(id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err = db.Exec("DELETE FROM todo WHERE id = ?", id)
	if err != nil {
		c.IndentedJSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, todo)
}

func main() {
	cfg := mysql.Config{
		User:   "admin",
		Passwd: "admin",
		// User:   os.Getenv("DBUSER"),
		// Passwd: os.Getenv("DBPASS"),
		Net:    "tcp",
		Addr:   "127.0.0.1:3306",
		DBName: "todo",
	}
	// Get a database handle.
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatal(err)
	}

	// Call DB.Ping to confirm that connecting to the database works. At run time,
	// sql.Open might not immediately connect, depending on the driver.
	// You’re using Ping here to confirm that the database/sql package can connect when it needs to.
	pingErr := db.Ping()
	if pingErr != nil {
		log.Fatal(pingErr)
	}
	fmt.Println("Connected!")

	router := gin.Default()
	router.GET("/todos", getTodos)
	router.POST("/todos", createTodo)
	router.GET("/todos/:id", getTodo)
	router.PATCH("/todos/:id", toggleTodoStatus)
	router.PUT("/todos/:id", updateTodo)
	router.DELETE("/todos/:id", deleteTodo)
	router.Run("localhost:9191")
}
