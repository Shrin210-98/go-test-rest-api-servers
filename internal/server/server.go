package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"example.com/tutorial/internal/database"
	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Service interface {
	// Health() map[string]string

	Close() error
	GetQueries() *database.Queries
}

type service struct {
	db      *sql.DB
	queries *database.Queries
}

var (
	dbName     = "postgres"
	username   = "postgres"
	password   = "admin"       //"postgres"
	port       = "9000"        //"8080"
	host       = "172.19.16.1" //"10.255.255.254" //"localhost"
	schema     = "public"
	dbInstance *service
)

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}
	// connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, dbName, schema)
	// db, err := sql.Open("pgx", connStr)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// dbInstance = &service{db: db}

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable search_path=%s", host, port, username, password, dbName, schema)
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		log.Fatal(err)
	}

	queries := database.New(conn)

	dbInstance = &service{db: nil, queries: queries}
	return dbInstance
}

func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", dbName)
	return s.db.Close()
}

func (s *service) GetQueries() *database.Queries {
	return s.queries
}

type Server struct {
	port int
	db   Service
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(port)
	NewServer := &Server{
		port: port,
		db:   New(),
	}

	middlewareChain := MiddlewareChain(RequestLoggerMiddleware, RequireAuthMiddleware)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      middlewareChain(NewServer.RegisterRoutes()),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.HelloWorldHandler)
	mux.HandleFunc("GET /authors", s.GetAuthorsHandler)
	mux.HandleFunc("POST /authors", s.PostAuthorsHandler)
	mux.HandleFunc("DELETE /authors/{id}", s.DeleteAuthorHandler)
	return mux
}

func (s *Server) HelloWorldHandler(w http.ResponseWriter, r *http.Request) {
	resp := make(map[string]string)
	resp["message"] = "Hello World"

	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Fatalf("error handling JSON marshal. Err: %v", err)
	}
	_, _ = w.Write(jsonResp)
}

func (s *Server) GetAuthorsHandler(w http.ResponseWriter, r *http.Request) {

	queries := s.db.GetQueries()
	authors, err := queries.ListAuthors(r.Context())
	if err != nil {
		http.Error(w, "could not fetch authors", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(authors)
}

func (s *Server) PostAuthorsHandler(w http.ResponseWriter, r *http.Request) {

	// Parse request body
	// type AuthorInput struct {
	// 	Name string `json:"name"`
	// 	Bio  string `json:"bio"`
	// }
	// var ainp AuthorInput
	// err := json.NewDecoder(r.Body).Decode(&ainp)
	// queries := s.db.GetQueries()
	// author, err := queries.CreateAuthor(r.Context(), database.CreateAuthorParams{
	// 	Name: ainp.Name,
	// 	Bio:  pgtype.Text{String: ainp.Bio, Valid: true},
	// })
	var authorData database.CreateAuthorParams
	if err := json.NewDecoder(r.Body).Decode(&authorData); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	author, err := s.db.GetQueries().CreateAuthor(r.Context(), authorData)

	if err != nil {
		http.Error(w, "Failed to create author", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(author)
}

func (s *Server) DeleteAuthorHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	queries := s.db.GetQueries()
	err = queries.DeleteAuthor(r.Context(), int64(id))
	if err != nil {
		http.Error(w, "could not delete author", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
