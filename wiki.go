package main

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
)

// Page is the wiki page content
type Page struct {
	Title string
	Body  []byte
}

var (
	viewPath     = "/view/"
	editPath     = "/edit/"
	savePath     = "/save/"
	contentPath  = "/content/"
	templatePath = "/view/"
)

var validPath = regexp.MustCompile("^/(edit|save|view)/([a-zA-Z0-9]+)$")
var templates *template.Template

func getFilepath(filename, path string) (string, error) {
	workingDirectory, err := os.Getwd()

	if err != nil {
		return "", err
	}

	return filepath.Join(workingDirectory, path, filename), nil
}

func initTemplate(names ...string) (*template.Template, error) {
	var templateFiles []string

	for _, name := range names {
		path, err := getFilepath(name+".html", templatePath)
		if err != nil {
			return nil, err
		}
		templateFiles = append(templateFiles, path)
	}

	return template.ParseFiles(templateFiles...)
}

func (page *Page) save() error {
	path, err := getFilepath(page.Title+".txt", contentPath)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, page.Body, 0600)
}

func loadPage(title string) (*Page, error) {
	filename, filenameError := getFilepath(title+".txt", contentPath)
	if filenameError != nil {
		return nil, filenameError
	}

	body, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, err
	}

	return &Page{Title: title, Body: body}, nil
}

// omits the leading route in the url (ex: for url ="/view/home" this function returns "home")
func getTitle(request *http.Request) (string, error) {
	//for match = [url route title] the tile will be match[2]
	match := validPath.FindStringSubmatch(request.URL.Path)
	if match == nil || len(match) < 3 {
		return "", errors.New("invalid page title")
	}
	return match[2], nil
}

func renderTemplate(responseWriter http.ResponseWriter, templateFileName string, page *Page) {
	err := templates.ExecuteTemplate(responseWriter, templateFileName+".html", page)
	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
	}
}

func makeHandler(handler func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		title, titleError := getTitle(request)

		if titleError != nil {
			http.NotFound(responseWriter, request)
			return
		}

		handler(responseWriter, request, title)
	}
}

func viewHandler(responseWriter http.ResponseWriter, request *http.Request, title string) {
	page, err := loadPage(title)

	if err != nil {
		http.Redirect(responseWriter, request, editPath+title, http.StatusFound)
		return
	}

	renderTemplate(responseWriter, "view", page)

}

func editHandler(responseWriter http.ResponseWriter, request *http.Request, title string) {
	page, err := loadPage(title)

	if err != nil {
		page = &Page{Title: title}
	}

	renderTemplate(responseWriter, "edit", page)
}

func saveHandler(responseWriter http.ResponseWriter, request *http.Request, title string) {
	body := request.FormValue("body")

	page := Page{Title: title, Body: []byte(body)}
	err := page.save()

	if err != nil {
		http.Error(responseWriter, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(responseWriter, request, viewPath+page.Title, http.StatusFound)
}

func init() {
	templates = template.Must(initTemplate("view", "edit"))
}

func main() {
	port := "8080"
	http.HandleFunc(viewPath, makeHandler(viewHandler))
	http.HandleFunc(editPath, makeHandler(editHandler))
	http.HandleFunc(savePath, makeHandler(saveHandler))
	http.HandleFunc("/", http.NotFound)
	fmt.Println("Listening on port ", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
