package utils

import (
    "log"
    "time"
	"fmt"
	"database/sql"

    "task-api/models"
)

// AddTask inserts a new task into the database
func AddTask(title, description string, position int) (*models.Task, error) {
    query := `
    INSERT INTO tasks (title, description, position, created_at, updated_at)
    VALUES (?, ?, ?, ?, ?)
    `
    result, err := db.Exec(query, title, description, position, time.Now(), time.Now())
    if err != nil {
        return nil, err
    }

    id, err := result.LastInsertId()
    if err != nil {
        return nil, err
    }

    return &models.Task{
        ID:          int(id),
        Title:       title,
        Description: description,
        Position:    position,
        CreatedAt:   time.Now(),
        UpdatedAt:   time.Now(),
    }, nil
}

func AddTasks(tasks []models.Task) ([]models.Task, error) {
    var createdTasks []models.Task

    for _, task := range tasks {
        newTask, err := AddTask(task.Title, task.Description, task.Position)
        if err != nil {
            return nil, err
        }
        createdTasks = append(createdTasks, *newTask)
    }

    return createdTasks, nil
}

// GetAllTasks retrieves all tasks sorted by position
func GetAllTasks() ([]models.Task, error) {
    query := `
    SELECT id, title, description, position, created_at, updated_at
    FROM tasks
    ORDER BY position ASC
    `
    rows, err := db.Query(query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var tasks []models.Task
    for rows.Next() {
        var task models.Task
        err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Position, &task.CreatedAt, &task.UpdatedAt)
        if err != nil {
            log.Println(err)
            continue
        }
        tasks = append(tasks, task)
    }
    return tasks, nil
}

// GetPaginatedTasks retrieves a subset of tasks and the total count
func GetPaginatedTasks(offset, limit int) ([]models.Task, int, error) {
    var tasks []models.Task
    var totalCount int

    // Count total tasks
    countQuery := "SELECT COUNT(*) FROM tasks"
    err := db.QueryRow(countQuery).Scan(&totalCount)
    if err != nil {
        return nil, 0, err
    }

    // Fetch paginated tasks
    query := `
    SELECT id, title, description, position, created_at, updated_at
    FROM tasks
    ORDER BY position ASC
    LIMIT ? OFFSET ?
    `
    rows, err := db.Query(query, limit, offset)
    if err != nil {
        return nil, 0, err
    }
    defer rows.Close()

    for rows.Next() {
        var task models.Task
        err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Position, &task.CreatedAt, &task.UpdatedAt)
        if err != nil {
            return nil, 0, err
        }
        tasks = append(tasks, task)
    }

    return tasks, totalCount, nil
}

// UpdateTask updates an existing task
func UpdateTask(id int, title, description string, position int) error {
    query := `
    UPDATE tasks
    SET title = ?, description = ?, position = ?, updated_at = ?
    WHERE id = ?
    `
    _, err := db.Exec(query, title, description, position, time.Now(), id)
    return err
}

// DeleteTask deletes a task by ID
func DeleteTask(id int) error {
    query := `
    DELETE FROM tasks WHERE id = ?
    `
    _, err := db.Exec(query, id)
    return err
}

// ReorderTasks updates the positions of multiple tasks
func ReorderTasks(updatedTasks []models.Task) error {
    tx, err := db.Begin()
    if err != nil {
        return err
    }

    query := `
    UPDATE tasks
    SET position = ?, updated_at = ?
    WHERE id = ?
    `
    for _, task := range updatedTasks {
        if task.ID == 0 || task.Position <= 0 {
            tx.Rollback()
            return fmt.Errorf("invalid task data: ID and position are required")
        }
        _, err := tx.Exec(query, task.Position, time.Now(), task.ID)
        if err != nil {
            tx.Rollback()
            return err
        }
    }

    return tx.Commit()
}

// CheckTaskExists checks if a task with the given ID exists
func CheckTaskExists(id int) (bool, error) {
    var count int
    query := "SELECT COUNT(*) FROM tasks WHERE id = ?"
    err := db.QueryRow(query, id).Scan(&count)
    if err != nil {
        return false, err
    }
    return count > 0, nil
}

// DeleteAllTasks deletes all tasks from the database
func DeleteAllTasks() error {
    query := "DELETE FROM tasks"
    _, err := db.Exec(query)
    return err
}

// GenerateDummyTasks inserts a specified number of dummy tasks into the database
func GenerateDummyTasks(count int) error {
    log.Printf("Starting to generate %d dummy tasks", count)

    lastTask, err := GetLastTask()
    if err != nil {
        log.Printf("Error fetching last task: %v", err)
        return err
    }

    startID := 1
    startPosition := 1
    if lastTask != nil {
        startID = lastTask.ID + 1
        startPosition = lastTask.Position + 1
    }

    log.Printf("Starting ID: %d, Starting Position: %d", startID, startPosition)

    // Optimize SQLite settings
    _, err = db.Exec("PRAGMA synchronous = OFF")
    if err != nil {
        log.Printf("Error setting PRAGMA synchronous = OFF: %v", err)
        return err
    }
    _, err = db.Exec("PRAGMA journal_mode = MEMORY")
    if err != nil {
        log.Printf("Error setting PRAGMA journal_mode = MEMORY: %v", err)
        return err
    }
    _, err = db.Exec("PRAGMA cache_size = -10000") // Allocate 10MB of cache
    if err != nil {
        log.Printf("Error setting PRAGMA cache_size = -10000: %v", err)
        return err
    }

    stmt, err := db.Prepare(`
        INSERT INTO tasks (title, description, position, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?)
    `)
    if err != nil {
        log.Printf("Error preparing INSERT statement: %v", err)
        return err
    }
    defer stmt.Close()

    batchSize := 1000 
    for i := 0; i < count; i++ {
        id := startID + i
        position := startPosition + i
        title := fmt.Sprintf("Task %d", id)
        description := fmt.Sprintf("Description for task %d", id)
        createdAt := time.Now()
        updatedAt := createdAt

        if i%batchSize == 0 {
            log.Printf("Starting batch %d-%d", id, id+batchSize-1)
            tx, txErr := db.Begin()
            if txErr != nil {
                log.Printf("Error starting transaction: %v", txErr)
                return txErr
            }
            defer func() {
                if txErr != nil {
                    log.Printf("Rolling back transaction due to error: %v", txErr)
                    tx.Rollback()
                } else {
                    log.Println("Committing transaction")
                    tx.Commit()
                }
            }()
        }

        _, execErr := stmt.Exec(title, description, position, createdAt, updatedAt)
        if execErr != nil {
            log.Printf("Error inserting task %d: %v", id, execErr)
            return execErr
        }
    }

    log.Printf("Successfully inserted %d dummy tasks", count)
    return nil
}

// GetLastTask retrieves the last inserted task
func GetLastTask() (*models.Task, error) {
    var task models.Task
    query := `
    SELECT id, title, description, position, created_at, updated_at
    FROM tasks
    ORDER BY id DESC
    LIMIT 1
    `
    err := db.QueryRow(query).Scan(&task.ID, &task.Title, &task.Description, &task.Position, &task.CreatedAt, &task.UpdatedAt)
    if err == sql.ErrNoRows {
        return nil, nil 
    }
    if err != nil {
        return nil, err
    }
    return &task, nil
}

// CheckPositionExists checks if a position already exists in the database
func CheckPositionExists(position int) (bool, error) {
    var count int
    query := "SELECT COUNT(*) FROM tasks WHERE position = ?"
    err := db.QueryRow(query, position).Scan(&count)
    if err != nil {
        return false, err
    }
    return count > 0, nil
}