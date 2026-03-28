package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

type Task struct {
	ID    uuid.UUID `json:"id"`
	Title string    `json:"title"`
	Done  bool      `json:"done"`
}

type CreateTaskRequest struct {
	Title string `json:"title"`
}

type UpdateTaskRequest struct {
	Done  *bool   `json:"done"`
	Title *string `json:"title"`
}

type UpdateTaskBody struct {
	Task UpdateTaskRequest `json:"task"`
}

type CreateTaskBody struct {
	Task CreateTaskRequest `json:"task"`
}

func main() {
	http.HandleFunc("/tasks", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":

			listTasks(w)
		case "POST":
			var newTask CreateTaskBody
			decoderError := json.NewDecoder(r.Body).Decode(&newTask)

			if decoderError != nil {
				http.Error(w, "Error", http.StatusBadRequest)

				return
			}

			createTask(w, newTask)
		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	})

	http.HandleFunc("/tasks/", func(w http.ResponseWriter, r *http.Request) {
		taskId := strings.TrimPrefix(r.URL.Path, "/tasks/")

		switch r.Method {
		case "DELETE":
			deleteTask(w, taskId)
		case "PUT":
			var updateTaskBody UpdateTaskBody
			decoderError := json.NewDecoder(r.Body).Decode(&updateTaskBody)

			if decoderError != nil {
				http.Error(w, "Error "+decoderError.Error(), http.StatusBadRequest)

				return
			}

			updateTask(w, taskId, updateTaskBody)
		}
	})

	port := ":8080"

	fmt.Printf("Server started on port %s\n", port)

	http.ListenAndServe(port, nil)
}

func loadTasks() ([]Task, error) {
	fmt.Println("Loading tasks...")

	file, error := os.ReadFile("./tasks.json")

	if error != nil {
		fmt.Printf("Error: %v\nCreating...\n", error)

		os.WriteFile("./tasks.json", []byte("[]"), 0666)
	}

	file, error = os.ReadFile("./tasks.json")

	var tasks []Task
	error = json.Unmarshal(file, &tasks)

	if error != nil {
		fmt.Printf("Could not json parse tasks.json. Reason: %v\n", error)

		return nil, error
	}

	fmt.Println("Successfully loaded tasks")

	return tasks, nil
}

// GET    /tasks          → list all tasks
func listTasks(w http.ResponseWriter) {
	loadedTasks := preloadTasks()

	jsonData, error := json.Marshal(loadedTasks)

	if error != nil {
		fmt.Printf("Error: %v", error)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// POST   /tasks          → create a task
func createTask(w http.ResponseWriter, newTask CreateTaskBody) {
	loadedTasks := preloadTasks()

	task := Task{
		ID:    uuid.New(),
		Title: newTask.Task.Title,
		Done:  false,
	}

	loadedTasks = append(loadedTasks, task)
	jsonData, error := json.Marshal(loadedTasks)

	if error != nil {
		http.Error(w, "Error adding task. Reason: "+error.Error(), http.StatusInternalServerError)

		return
	}

	jsonTask, error := json.Marshal(task)

	if error != nil {
		http.Error(w, "Error adding task. Reason: "+error.Error(), http.StatusInternalServerError)

		return
	}

	os.WriteFile("./tasks.json", jsonData, 0666)

	w.WriteHeader(http.StatusOK)
	w.Write(jsonTask)
}

// DELETE /tasks/{id}     → delete a task
func deleteTask(w http.ResponseWriter, taskId string) {
	loadedTasks := preloadTasks()

	taskIndex, _, error := lookupTaskByUuid(loadedTasks, taskId)

	if error != nil {
		http.Error(w, "No task found with uuid: "+taskId, http.StatusNotFound)

		return
	}

	loadedTasks = append(loadedTasks[:taskIndex], loadedTasks[taskIndex+1:]...)

	jsonData, _ := json.Marshal(loadedTasks)

	os.WriteFile("./tasks.json", jsonData, 0666)

	w.WriteHeader(http.StatusNoContent)
}

func preloadTasks() []Task {
	loadedTasks, error := loadTasks()

	if error != nil {
		fmt.Println("Error loading tasks. ", error)

		return []Task{}
	}

	return loadedTasks
}

func lookupTaskByUuid(tasks []Task, uuid string) (int, Task, error) {
	for index, element := range tasks {
		if uuid == element.ID.String() {
			return index, element, nil
		}
	}

	return -1, Task{}, errors.New("Task not found")
}

func updateTask(w http.ResponseWriter, taskId string, updateTaskBody UpdateTaskBody) {
	loadedTasks := preloadTasks()

	taskIndex, _, error := lookupTaskByUuid(loadedTasks, taskId)

	if error != nil {
		http.Error(w, "No task found with uuid: "+taskId, http.StatusNotFound)

		return
	}

	if updateTaskBody.Task.Done != nil {
		loadedTasks[taskIndex].Done = *updateTaskBody.Task.Done
	}

	if updateTaskBody.Task.Title != nil {
		loadedTasks[taskIndex].Title = *updateTaskBody.Task.Title
	}

	jsonData, error := json.Marshal(loadedTasks)

	if error != nil {
		http.Error(w, "Failed to update task: "+taskId, http.StatusInternalServerError)

		return
	}

	taskJson, error := json.Marshal(loadedTasks[taskIndex])

	if error != nil {
		http.Error(w, "Failed to update task: "+taskId, http.StatusInternalServerError)

		return
	}

	os.WriteFile("./tasks.json", jsonData, 0666)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(taskJson)
}
