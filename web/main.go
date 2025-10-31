package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"dnd-char-generator/internal/application"
	"dnd-char-generator/internal/infrastructure"
	"dnd-char-generator/internal/infrastructure/dndapi"
	"dnd-char-generator/internal/infrastructure/persistence"
)

type Server struct {
	Service *application.CharacterService
}

var templates *template.Template

const listHTML = `
<!DOCTYPE html>
<html>
<head><title>Character List</title></head>
<body>
    <h1>D&D Character List</h1>
    <ul>
    {{range .}}
        <li><a href="/characters/{{.Name}}">{{.Name}} (Lvl {{.Level}} {{.Class}})</a></li>
    {{end}}
    </ul>
</body>
</html>`

func init() {
	templates = template.Must(template.New("list").Parse(listHTML))

	var err error
	templates, err = templates.ParseFiles("web/templates/charactersheet.html")
	if err != nil {
		log.Fatalf("FATAL: Could not load charactersheet.html template. Ensure the file is at web/templates/charactersheet.html. Error: %v", err)
	}
}

func initApp() (*application.CharacterService, error) {
	allSpells, allWeapons, allArmors, allShields, err := infrastructure.LoadData("5e-SRD-Equipment.csv", "5e-SRD-Spells.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to load static SRD data: %w", err)
	}

	repo := persistence.NewFileRepository("characters.json")
	apiClient := dndapi.NewClient()
	service := application.NewCharacterService(repo, apiClient, allSpells, allWeapons, allArmors, allShields)

	return service, nil
}

func main() {
	service, err := initApp()
	if err != nil {
		os.Exit(1)
	}

	app := &Server{
		Service: service,
	}

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	mux.HandleFunc("GET /characters", app.listCharactersHandler)
	mux.HandleFunc("GET /characters/{name}", app.viewCharacterHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)

	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("Starting web server on %s", addr)
	err = srv.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func (app *Server) listCharactersHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	characters, err := app.Service.ListCharacters(ctx)
	if err != nil {
		log.Printf("ERROR: Failed to fetch characters: %v", err)
		http.Error(w, "Failed to retrieve character list.", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "list", characters); err != nil {
		log.Printf("ERROR: Failed to execute list template: %v", err)
		http.Error(w, "Failed to render list template", http.StatusInternalServerError)
	}
}

func (app *Server) viewCharacterHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "Character name is required in the path.", http.StatusBadRequest)
		return
	}

	charName := strings.ReplaceAll(name, "%20", " ")

	ctx := context.Background()

	char, err := app.Service.GetCharacter(ctx, charName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, fmt.Sprintf("Character '%s' not found.", charName), http.StatusNotFound)
			return
		}

		log.Printf("ERROR: Failed to fetch character '%s' (Enrichment likely failed): %v", charName, err)
		http.Error(w, "Failed to retrieve character sheet (check logs for API error).", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, "charactersheet.html", char); err != nil {
		log.Printf("ERROR: Failed to execute sheet template: %v", err)
		http.Error(w, "Failed to render character sheet template", http.StatusInternalServerError)
	}
}
