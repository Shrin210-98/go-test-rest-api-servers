package main

import (
	"context"
	"log"
	"reflect"

	"example.com/tutorial/internal/database"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func run() error {
	ctx := context.Background()

	// conn, err := pgx.Connect(ctx, "user=pqgotest dbname=pqgotest sslmode=verify-full")
	conn, err := pgx.Connect(ctx, "host=localhost user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	queries := database.New(conn)

	// list all authors
	authors, err := queries.ListAuthors(ctx)
	if err != nil {
		return err
	}
	log.Println(authors)

	// create an author
	// insertedAuthor, err := queries.CreateAuthor(ctx, database.CreateAuthorParams{
	// 	Name: "Brian Kernighan",
	// 	Bio:  pgtype.Text{String: "Co-author of The C Programming Language and The Go Programming Language", Valid: true},
	// })

	insertedAuthor, err := queries.CreateAuthor(ctx, database.CreateAuthorParams{
		Name: "NaMo",
		Bio:  pgtype.Text{String: "Prime Minister of India", Valid: true},
	})
	if err != nil {
		return err
	}
	log.Println(insertedAuthor)

	// get the author we just inserted
	fetchedAuthor, err := queries.GetAuthor(ctx, insertedAuthor.ID)
	if err != nil {
		return err
	}

	// prints true
	log.Println(reflect.DeepEqual(insertedAuthor, fetchedAuthor))
	return nil
}

func main_v1() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
