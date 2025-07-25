package serverv2

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"

	"example.com/tutorial/internal/database"
	"example.com/tutorial/internal/utils"
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
	password   = "admin"       // LMDE "postgres"
	port       = "9000"        //"8080"
	host       = "172.19.16.1" //"localhost"
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
	log.Printf("Connecting to server 2...")
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

	mux := NewServer.RegisterRoutes()
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if (path == "/register" || path == "/login") && r.Method == http.MethodPost {
			mux.ServeHTTP(w, r)
			return
		}
		// if && r.Method == http.MethodPost {
		// 	mux.ServeHTTP(w, r)
		// 	return
		// }
		MiddlewareChain(RequestLoggerMiddleware, RequireAuthMiddleware)(mux).ServeHTTP(w, r)
	})
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      handler,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	return server
}

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.HelloWorldHandler)
	mux.HandleFunc("GET    /authors", s.GetAuthorsHandler)
	mux.HandleFunc("DELETE /authors/{id}", s.DeleteAuthorHandler)

	mux.HandleFunc("POST   /register", s.RegisterHandler)
	mux.HandleFunc("POST   /login", s.LoginHandler)
	mux.HandleFunc("GET    /logout", s.HelloWorldHandler)
	mux.HandleFunc("POST   /protected", s.HelloWorldHandler)
	return mux
}

type Login struct {
	HashedPassword string
	SessionToken   string
	CSRFToken      string
}

var users = map[string]Login{}

func (s *Server) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	// username := r.FormValue("username")
	// password := r.FormValue("password")
	// fmt.Printf("Username %s and Password %s and R = r", username, password)
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(req.Username) < 8 || len(req.Password) < 8 {
		http.Error(w, "Invalid Username or Password", http.StatusNotAcceptable)
		return
	}
	if _, ok := users[req.Username]; ok {
		http.Error(w, "User Already Exists", http.StatusConflict)
		return
	}

	hashedPassword, _ := utils.HashPassword(req.Password)
	users[req.Username] = Login{
		HashedPassword: hashedPassword,
	}
	fmt.Fprintln(w, "User Registered Successfully")

}

func (s *Server) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	user, ok := users[req.Username]

	fmt.Println("User ", users, user, user.HashedPassword, utils.CheckPasswordHash(req.Password, user.HashedPassword))

	if !ok || !utils.CheckPasswordHash(req.Password, user.HashedPassword) {
		http.Error(w, "Invalid Username or Password", http.StatusUnauthorized)
		return
	}

	sessionToken := utils.GenerateToken(32)
	csrfToken := utils.GenerateToken(32)

	//Set Session Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})

	//Set CSRF Token in a Cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    csrfToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: false,
	})

	user.SessionToken = sessionToken
	user.CSRFToken = csrfToken
	users[req.Username] = user
	fmt.Fprintln(w, "Login Successful")
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

func (s *Server) DeleteAuthorHandler(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		// http.Error(w, "Invalid ID", http.StatusBadRequest)
		utils.JsonResponseMsg(w, http.StatusBadRequest, map[string]string{"message": "Invalid ID"})
		return
	}

	queries := s.db.GetQueries()
	err = queries.DeleteAuthor(r.Context(), int64(id))
	if err != nil {
		http.Error(w, "could not delete author", http.StatusInternalServerError)
		return
	}

	// w.WriteHeader(http.StatusNoContent)
	// resp := make(map[string]string)
	// resp["message"] = "Successfully Deleted Author"
	// jsonResp, err := json.Marshal(resp)
	// if err != nil {
	// 	log.Fatalf("error handling JSON marshal. Err: %v", err)
	// }
	// _, _ = w.Write(jsonResp)

	utils.JsonResponseMsg(w, http.StatusOK, map[string]string{"message": "Successfully Deleted Author"})
}
