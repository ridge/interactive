package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/ridge/must"
	"github.com/spf13/pflag"
)

func cors(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("access-control-allow-origin", "*")
		w.Header().Set("access-control-allow-methods", "GET, POST, PATCH, DELETE")
		w.Header().Set("access-control-allow-headers", "accept, content-type")
		if r.Method == "OPTIONS" {
			return // Preflight sets headers and we're done
		}
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func contentTypeJsonHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func commonHandlers(next http.HandlerFunc) http.Handler {
	return contentTypeJsonHandler(cors(next))
}

type TODOService interface {
	GetAll(context.Context) ([]TODO, error)
	Get(ctx context.Context, id int) (*TODO, error)
	Save(ctx context.Context, todo TODO) error
	DeleteAll(context.Context) error
	Delete(ctx context.Context, id int) error
}

func main() {
	var listenAddr string
	var postgresqlURL string
	var boltDBFile string

	pflag.StringVar(&listenAddr, "listen", ":44444", "listen address")
	pflag.StringVar(&postgresqlURL, "postgresql-url", "", "PostgreSQL URL to connect")
	pflag.StringVar(&boltDBFile, "boltdb-file", "", "BoltDB file")
	pflag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("Backend starting...")

	var svc TODOService
	var err error

	switch {
	case postgresqlURL != "" && boltDBFile != "":
		panic("--postgresql-url and --boltdb-file are exclusive")
	case postgresqlURL == "" && boltDBFile == "":
		panic("One of --postgresql-url and --boltdb-file is required")
	case postgresqlURL != "":
		svc, err = NewPostgreSQLTODOService(ctx, postgresqlURL)
	default:
		svc, err = NewBoltDBTODOService(boltDBFile)
	}
	must.OK(err)

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigs
		fmt.Printf("Received %s, shutting down...\n", sig)
		cancel()
	}()

	mux := http.NewServeMux()

	h := commonHandlers(func(w http.ResponseWriter, r *http.Request) {
		todoHandler(r.Context(), svc, w, r)
	})

	mux.Handle("/todos", h)
	mux.Handle("/todos/", h)

	server := http.Server{
		Handler:     mux,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	listener := must.NetListener(net.Listen("tcp", listenAddr))

	fmt.Printf("Backend is listening on %s\n", listenAddr)

	go func() {
		<-ctx.Done()
		listener.Close()
	}()

	server.Serve(listener)
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

func todoHandler(ctx context.Context, svc TODOService, w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	key := ""
	if len(parts) > 2 {
		key = parts[2]
	}

	switch r.Method {
	case "GET":
		if len(key) == 0 {
			todos, err := svc.GetAll(ctx)
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
			todo, err := svc.Get(ctx, id)
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
		err = svc.Save(ctx, todo)
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

		err = svc.Save(ctx, todo)
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
			svc.DeleteAll(ctx)
		} else {
			id, err := strconv.Atoi(key)
			if err != nil {
				http.Error(w, "Invalid Id", http.StatusBadRequest)
				return
			}
			err = svc.Delete(ctx, id)
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
