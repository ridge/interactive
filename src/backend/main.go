package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/ridge/must"
	"github.com/spf13/pflag"
)

type TODOService interface {
	GetAll() ([]TODO, error)
	Get(id int) (*TODO, error)
	Save(todo *TODO) error
	DeleteAll() error
	Delete(id int) error
}

func main() {
	var listenAddr string
	var postgresqlURL string
	var boltDBFile string

	pflag.StringVar(&listenAddr, "listen", ":44444", "listen address")
	pflag.StringVar(&postgresqlURL, "postgresql-url", "", "PostgreSQL URL to connect")
	pflag.StringVar(&boltDBFile, "boltbdb-file", "", "BoltDB file")

	pflag.Parse()

	var svc TODOService
	var err error

	switch {
	case postgresqlURL != "" && boltDBFile != "":
		panic("--postgresql-url and --boltdb-file are exclusive")
	case postgresqlURL == "" && boltDBFile == "":
		panic("One of --postgresql-url and --boltdb-file is required")
	case postgresqlURL != "":
		svc, err = NewPostgreSQLTODOService(postgresqlURL)
	default:
		svc, err = NewBoltDBTODOService(boltDBFile)
	}
	must.OK(err)

	mux := http.NewServeMux()

	h := commonHandlers(func(w http.ResponseWriter, r *http.Request) {
		todoHandler(svc, w, r)
	})

	mux.Handle("/todos", h)
	mux.Handle("/todos/", h)

	log.Fatal(http.ListenAndServe(listenAddr, mux))
}

func addUrlToTodos(r *http.Request, todos ...*TODO) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseUrl := scheme + "://" + r.Host + "/todos/"

	for _, todo := range todos {
		todo.URL = baseUrl + strconv.Itoa(todo.ID)
	}
}

func todoHandler(svc TODOService, w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	key := ""
	if len(parts) > 2 {
		key = parts[2]
	}

	switch r.Method {
	case "GET":
		if len(key) == 0 {
			todos, err := svc.GetAll()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			//			addUrlToTodos(r, todos...)
			json.NewEncoder(w).Encode(todos)
		} else {
			id, err := strconv.Atoi(key)
			if err != nil {
				http.Error(w, "Invalid Id", http.StatusBadRequest)
				return
			}
			todo, err := svc.Get(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if todo == nil {
				http.NotFound(w, r)
				return
			}
			//			addUrlToTodos(r, todo)
			json.NewEncoder(w).Encode(todo)
		}
	case "POST":
		if len(key) > 0 {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		todo := TODO{
			Completed: false,
		}
		err := json.NewDecoder(r.Body).Decode(&todo)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		err = svc.Save(&todo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		//		addUrlToTodos(r, &todo)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(todo)
	case "PATCH":
		id, err := strconv.Atoi(key)
		if err != nil {
			http.Error(w, "Invalid Id", http.StatusBadRequest)
			return
		}
		var todo TODO
		err = json.NewDecoder(r.Body).Decode(&todo)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		todo.ID = id

		err = svc.Save(&todo)
		if err != nil {
			if strings.ToLower(err.Error()) == "not found" {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		//		addUrlToTodos(r, &todo)
		json.NewEncoder(w).Encode(todo)
	case "DELETE":
		if len(key) == 0 {
			svc.DeleteAll()
		} else {
			id, err := strconv.Atoi(key)
			if err != nil {
				http.Error(w, "Invalid Id", http.StatusBadRequest)
				return
			}
			err = svc.Delete(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}
