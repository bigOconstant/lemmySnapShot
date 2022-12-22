package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

type Post struct {
	CommunityName string
	PostName      string
	PostBody      sql.NullString
	PostUrl       sql.NullString
	PostThumbNail sql.NullString
	PostId        int
}
type Site struct {
	Name        string
	Description string
	SideBar     string
	Icon        string
	SiteUrl     string
	TopPost     Post
}

func (s Site) IsNotNull(t sql.NullString) bool {
	if t.Valid {
		return true
	}
	return false
}

const getMostActivePost = `select community.name as community_name, post.name as post_name,post.body as post_body,post.url as post_url,post.thumbnail_url as post_thumbnail, post.id as post_id from post_aggregates 
left join post on post_aggregates.post_id = post.id left join community on post.community_id = community.id where post.community_id in (select community.id from community_aggregates left join community on  community_aggregates.community_id = community.id order by community_aggregates.users_active_day desc, community_aggregates.comments desc limit 1
) order by post_aggregates.newest_comment_time desc limit 1`

func main() {
	godotenv.Load(".env")
	fmt.Println("Begin writing to html document")
	//siteUrl := "http://localhost:1236"
	siteUrl := os.Getenv("SITEURL")
	ctx := context.Background()

	// Set up connection
	conn, err := pgx.Connect(context.Background(), "postgresql://lemmy:password@localhost:5433/lemmy")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close(ctx)

	// Get Site Details
	site := Site{}
	err = conn.QueryRow(ctx, "select name,icon, description,sidebar from site order by published limit 1").Scan(&site.Name, &site.Icon, &site.Description, &site.SideBar)

	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}

	// Get Most active Post from the most Active community
	post := Post{}
	err = conn.QueryRow(ctx, getMostActivePost).Scan(&post.CommunityName, &post.PostName, &post.PostBody, &post.PostUrl, &post.PostThumbNail, &post.PostId)

	if err != nil {
		fmt.Fprintf(os.Stderr, "QueryRow failed: %v\n", err)
		os.Exit(1)
	}
	site.TopPost = post

	t, err := template.ParseFiles(os.Getenv("INPUTFILE"))
	if err != nil {
		log.Print(err)
		return
	}

	f, err := os.Create(os.Getenv("OUTPUTFILE"))
	if err != nil {
		log.Println("create file: ", err)
		return
	}
	site.SiteUrl = siteUrl
	err = t.Execute(f, site)
	if err != nil {
		fmt.Fprintf(os.Stderr, "template failed: %v\n", err)
		os.Exit(1)
	}
	f.Close()

}
