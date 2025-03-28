package main

import (
    "bytes"
	"encoding/json"
	"fmt"
    "io"
    "log"
    "net/http"
    "strconv"
	"time"

    "github.com/gin-gonic/gin"
    "task-api/models"
    "task-api/utils"
)

func main() {
    utils.InitDB()
    defer utils.CloseDB()

    r := gin.Default()

    // GET /tasks - List all tasks
    r.GET("/tasks", func(c *gin.Context) {
		pageStr := c.DefaultQuery("page", "1") 
		limitStr := c.DefaultQuery("limit", "10") 
	
		page, err := strconv.Atoi(pageStr)
		if err != nil || page <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid page value"})
			return
		}
	
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit value"})
			return
		}
	
		offset := (page - 1) * limit
	
		tasks, totalCount, err := utils.GetPaginatedTasks(offset, limit)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch tasks"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"_totalTasks": totalCount,
			"page":       page,
			"limit":      limit,
			"tasks":      tasks,
		})
	})

    // POST /tasks - Add a new task or an array of tasks
    r.POST("/tasks", func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		log.Printf("Received payload: %s", string(body))
	
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	
		var singleTask models.Task
		var multipleTasks []models.Task
	
		if err := json.Unmarshal(body, &multipleTasks); err == nil && len(multipleTasks) > 0 {
			log.Println("Decoded as an array of tasks")
			for _, task := range multipleTasks {
				if task.Title == "" || task.Position <= 0 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Title and position are required for all tasks"})
					return
				}
	
				exists, err := utils.CheckPositionExists(task.Position)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check position existence"})
					return
				}
				if exists {
					c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Position %d already exists", task.Position)})
					return
				}
			}
	
			// Add multiple tasks
			var createdTasks []models.Task
			for _, task := range multipleTasks {
				newTask, err := utils.AddTask(task.Title, task.Description, task.Position)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add tasks"})
					return
				}
				createdTasks = append(createdTasks, *newTask)
			}
			c.JSON(http.StatusCreated, createdTasks)
			return
		}
	
		if err := json.Unmarshal(body, &singleTask); err == nil {
			log.Println("Decoded as a single task")

			if singleTask.Title == "" || singleTask.Position <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Title and position are required"})
				return
			}
	
			exists, err := utils.CheckPositionExists(singleTask.Position)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check position existence"})
				return
			}
			if exists {
				c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Position %d already exists", singleTask.Position)})
				return
			}
	
			task, err := utils.AddTask(singleTask.Title, singleTask.Description, singleTask.Position)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add task"})
				return
			}
			c.JSON(http.StatusCreated, task)
			return
		}
	
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
	})

	// POST /tasks/generate?count=? - Generate tasks
	r.POST("/tasks/generate", func(c *gin.Context) {
		countStr := c.Query("count")
		if countStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Count parameter is required"})
			return
		}
	
		count, err := strconv.Atoi(countStr)
		if err != nil || count <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid count value"})
			return
		}
	
		err = utils.GenerateDummyTasks(count)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate dummy tasks"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Successfully generated %d dummy tasks", count)})
	})

    // PUT /tasks/:id - Update an existing task
    r.PUT("/tasks/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
			return
		}
	
		exists, err := utils.CheckTaskExists(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check task existence"})
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
	
		var input struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Position    int    `json:"position"`
		}
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
	
		err = utils.UpdateTask(id, input.Title, input.Description, input.Position)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update task"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Task updated successfully"})
	})

    // DELETE /tasks/:id - Delete a task
    r.DELETE("/tasks/:id", func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid task ID"})
			return
		}
	
		exists, err := utils.CheckTaskExists(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check task existence"})
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
			return
		}
	
		err = utils.DeleteTask(id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete task"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Task deleted successfully"})
	})

	// DELETE /tasks/:id - Delete all tasks
	r.DELETE("/tasks", func(c *gin.Context) {
		err := utils.DeleteAllTasks()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete all tasks"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "All tasks deleted successfully"})
	})

    // PATCH /tasks/reorder - Reorder tasks
    r.PATCH("/tasks/reorder", func(c *gin.Context) {
		var updatedTasks []struct {
			ID       int `json:"id"`
			Position int `json:"position"`
		}
		if err := c.ShouldBindJSON(&updatedTasks); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
	
		for _, task := range updatedTasks {
			if task.ID == 0 || task.Position <= 0 {
				c.JSON(http.StatusBadRequest, gin.H{"error": "ID and position are required for all tasks"})
				return
			}
	
			exists, err := utils.CheckTaskExists(task.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check task existence"})
				return
			}
			if !exists {
				c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("Task with ID %d not found", task.ID)})
				return
			}
		}
	
		tx, err := utils.GetDB().Begin()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start transaction"})
			return
		}
		defer tx.Rollback()
	
		query := "UPDATE tasks SET position = ?, updated_at = ? WHERE id = ?"
		for _, task := range updatedTasks {
			updatedAt := time.Now()
			_, err := tx.Exec(query, task.Position, updatedAt, task.ID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to update task with ID %d", task.ID)})
				return
			}
		}
	
		err = tx.Commit()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
			return
		}
	
		c.JSON(http.StatusOK, gin.H{"message": "Tasks reordered successfully"})
	})

    // Start the server
    r.Run(":3000")
}