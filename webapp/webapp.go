package webapp

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"example-api-server/appinterface"
)

type notFoundData struct {
	Page string
}

const notFoundTemplate = `<!DOCTYPE html>
<html>
	<head>
	  <title>Page not found</title>
	  <meta charset="UTF-8">
	</head>
	<body>
	  <h1>Page Not Found</h1>
	  <p>Could not find a page: {{ .Page }}</p>
	</body>
</html>
`

var (
	//go:embed pages
	pages         embed.FS
	wrapper       *template.Template
	errorTemplate *template.Template
	home          *template.Template
	notFound      *template.Template

	//go:embed js
	js embed.FS

	//go:embed css
	css embed.FS

	//go:embed img
	img embed.FS
)

// TODO: add support for the site prefix to JS and CSS resources
type fsHandler struct {
	webApp      *webApp
	fs          embed.FS
	typeHandler func(path string, response http.ResponseWriter)
}

func (h *fsHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	path := strings.TrimPrefix(request.URL.Path, "/")
	bts, err := h.fs.ReadFile(path)
	if err != nil {
		log.Printf("Error looking for %s: %v\n", path, err)
		h.webApp.notFound(path, response)
		return
	}
	h.typeHandler(request.URL.Path, response)
	_, err = response.Write(bts)
	if err != nil {
		log.Printf("Error sending response: %v\n", err)
	}
}

func (w *webApp) newFSHandler(fs embed.FS, typeHandler func(path string, response http.ResponseWriter)) http.Handler {
	return &fsHandler{
		webApp:      w,
		fs:          fs,
		typeHandler: typeHandler,
	}
}

func (w *webApp) newJSHandler() http.Handler {
	return w.newFSHandler(js, jsHeader)
}

func (w *webApp) newCSSHandler() http.Handler {
	return w.newFSHandler(css, cssHeader)
}

func (w *webApp) newImageHandler() http.Handler {
	return w.newFSHandler(img, imgHeader)
}

func initFromDir(parentPath string) {
	rawTemplates := template.Must(template.ParseFS(pages, parentPath+"/*.gohtml"))
	for _, tpl := range rawTemplates.Templates() {
		switch strings.TrimSuffix(filepath.Base(tpl.Name()), ".gohtml") {
		case "wrapper":
			wrapper = tpl
		case "home":
			home = tpl
		case "error":
			errorTemplate = tpl
		default:
			continue
		}
	}
}

func init() {
	initFromDir("pages")
	notFound = template.Must(template.New("not-found").Parse(notFoundTemplate))
}

type webApp struct {
	app appinterface.App
	mux *http.ServeMux
}

func (w *webApp) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	w.mux.ServeHTTP(writer, request)
}

func (w *webApp) setupRoutes() {
	w.mux.Handle("/js/*", w.newJSHandler())
	w.mux.Handle("/css/*", w.newCSSHandler())
	w.mux.Handle("/img/*", w.newImageHandler())
	w.mux.HandleFunc("GET /"+"{$}", w.renderIndex)
	w.mux.HandleFunc("GET /api/server-time", w.serverTime)
	w.mux.HandleFunc("POST /api/add-contact", w.addContact)
	w.mux.HandleFunc("GET /api/contacts", w.contacts)
	w.mux.HandleFunc("GET /api/contact/{id}", w.contact)
	w.mux.HandleFunc("PUT /api/contact/{id}", w.updateContact)
	w.mux.HandleFunc("DELETE /api/contact/{id}", w.deleteContact)
}

type serverTime struct {
	Time string `json:"time"`
}

type errorJson struct {
	Error string `json:"error"`
}

func (w *webApp) serverTime(response http.ResponseWriter, request *http.Request) {
	now := time.Now()
	data := serverTime{
		Time: now.Format(time.RFC3339),
	}
	w.sendJson(data, "Error marshalling time: %v", response)
}

func (w *webApp) addContact(response http.ResponseWriter, request *http.Request) {
	// Limit the size of the request body to 4KB
	// This is an example of protecting the server from overflow attacks
	request.Body = http.MaxBytesReader(response, request.Body, 4096)
	err := request.ParseMultipartForm(4096)
	if err != nil {
		log.Printf("Error: could not parse input form: %v", err)
		r := errorJson{
			Error: fmt.Sprintf("Error parsing form: %v", err),
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	firstName := request.Form.Get("firstName")
	lastName := request.Form.Get("lastName")
	email := request.Form.Get("email")
	// This is minimal validation.  Should check that the email looks valid.
	if firstName == "" || lastName == "" || email == "" {
		r := errorJson{
			Error: "Missing required fields",
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	// Notice that HTTP is a SERIALIZATION protocol.  One _must_ check for errors, and protect
	// against attack vectors like encoding "JOHNNY DROP TABLES" and the like. As well as making
	// sure that inputs are within expected ranges.  NEVER TRUST THE INTERNET!!!!
	// All the above boilerplate is because we cannot trust anything from the internet.
	err = w.app.AddContact(firstName, lastName, email) // <- This is how GOD intended it to be. ;-)
	if err != nil {
		r := errorJson{
			Error: fmt.Sprintf("Error adding contact: %v", err),
		}
		w.sendErrorJson(r, "Error marshalling error: %v", response)
		return
	}
}

func (w *webApp) contacts(response http.ResponseWriter, request *http.Request) {
	contacts, err := w.app.GetContacts()
	if err != nil {
		r := errorJson{
			Error: fmt.Sprintf("Error getting contacts: %v", err),
		}
		w.sendErrorJson(r, "Error marshalling error: %v", response)
		return
	}
	w.sendJson(contacts, "Error marshalling contacts: %v", response)
}

func (w *webApp) contact(response http.ResponseWriter, request *http.Request) {
	idString := request.PathValue("id")
	if idString == "" {
		r := errorJson{
			Error: "Missing ID",
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		r := errorJson{
			Error: fmt.Sprintf("Error parsing ID: %v", err),
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	if id <= 0 {
		r := errorJson{
			Error: "Invalid ID",
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	contact, err := w.app.ContactDetails(id)
	if err != nil {
		if err == io.EOF {
			r := errorJson{
				Error: "Contact not found",
			}
			w.sendStatusJson(r, http.StatusNotFound, "Error marshalling error: %v", response)
			return
		}
		r := errorJson{
			Error: fmt.Sprintf("Error getting contact: %v", err),
		}
		w.sendErrorJson(r, "Error marshalling error: %v", response)
		return
	}
	w.sendJson(contact, "Error marshalling contact: %v", response)
}

func (w *webApp) updateContact(response http.ResponseWriter, request *http.Request) {
	// Limit the size of the request body to 4KB
	// This is an example of protecting the server from overflow attacks
	request.Body = http.MaxBytesReader(response, request.Body, 4096)
	err := request.ParseForm()
	if err != nil {
		log.Printf("Error: could not parse input form: %v", err)
		r := errorJson{
			Error: fmt.Sprintf("Error parsing form: %v", err),
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	idString := request.PathValue("id")
	if idString == "" {
		r := errorJson{
			Error: "Missing ID",
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		r := errorJson{
			Error: fmt.Sprintf("Error parsing ID: %v", err),
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	if id <= 0 {
		r := errorJson{
			Error: "Invalid ID",
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	firstName := request.Form.Get("firstName")
	lastName := request.Form.Get("lastName")
	email := request.Form.Get("email")
	// This is minimal validation.  Should check that the email looks valid.
	if firstName == "" || lastName == "" || email == "" {
		r := errorJson{
			Error: "Missing required fields",
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	err = w.app.UpdateContact(id, firstName, lastName, email) // <- This is how GOD intended it to be. ;-)
	if err != nil {
		r := errorJson{
			Error: fmt.Sprintf("Error updating contact: %v", err),
		}
		w.sendErrorJson(r, "Error marshalling error: %v", response)
		return
	}
}

func (w *webApp) deleteContact(response http.ResponseWriter, request *http.Request) {
	idString := request.PathValue("id")
	if idString == "" {
		r := errorJson{
			Error: "Missing ID",
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	id, err := strconv.Atoi(idString)
	if err != nil {
		r := errorJson{
			Error: fmt.Sprintf("Error parsing ID: %v", err),
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	if id <= 0 {
		r := errorJson{
			Error: "Invalid ID",
		}
		w.sendStatusJson(r, http.StatusBadRequest, "Error marshalling error: %v", response)
		return
	}
	err = w.app.DeleteContact(id)
	if err != nil {
		r := errorJson{
			Error: fmt.Sprintf("Error deleting contact: %v", err),
		}
		w.sendErrorJson(r, "Error marshalling error: %v", response)
		return
	}
}

func NewWebApp(app appinterface.App) http.Handler {
	r := &webApp{
		app: app,
		mux: http.NewServeMux(),
	}
	r.setupRoutes()
	return r
}

type wrapperData struct {
	Title string
	Body  template.HTML
}

func templateToString(t *template.Template, data any) string {
	b := bytes.Buffer{}
	err := t.Execute(&b, data)
	if err != nil {
		return fmt.Sprintf("ERROR: %v", err)
	}
	return b.String()
}

func templateToHTML(t *template.Template, data any) template.HTML {
	return template.HTML(templateToString(t, data))
}

func renderError(err error) template.HTML {
	data := struct {
		Error string
	}{err.Error()}
	return templateToHTML(errorTemplate, &data)
}

func renderStringError(err string) template.HTML {
	data := struct {
		Error string
	}{err}
	return templateToHTML(errorTemplate, &data)
}

func redirectTo(url string, response http.ResponseWriter, request *http.Request) {
	http.Redirect(response, request, url, 301)
}

func standardHeaders(contentType string, response http.ResponseWriter) {
	response.Header().Add("Content-Type", contentType)
	response.Header().Add("Cache-Control", "no-cache")
}

func htmlHeader(response http.ResponseWriter) {
	standardHeaders("text/html", response)
}

func jsonHeader(path string, response http.ResponseWriter) {
	standardHeaders("application/json", response)
}

func jsHeader(path string, response http.ResponseWriter) {
	standardHeaders("text/javascript", response)
}

func cssHeader(path string, response http.ResponseWriter) {
	standardHeaders("text/css", response)
}

func imgHeader(path string, response http.ResponseWriter) {
	switch filepath.Ext(path) {
	case ".png":
		standardHeaders("image/png", response)
	case ".jpg", ".jpeg":
		standardHeaders("image/jpeg", response)
	case ".gif":
		standardHeaders("image/gif", response)
	case ".svg":
		standardHeaders("image/svg+xml", response)
	case ".ico":
		standardHeaders("image/vnd.microsoft.icon", response)
	default:
		standardHeaders("application/octet-stream", response)
	}
}

func (w *webApp) errorPage(title string, body template.HTML, status int, response http.ResponseWriter) {
	id := wrapperData{
		Title: title,
		Body:  body,
	}
	log.Println(string(body))
	response.WriteHeader(status)
	htmlHeader(response)
	err := wrapper.Execute(response, &id)
	if err != nil {
		log.Printf("Error generating page: %v\n", err)
	}
}

func (w *webApp) notFound(path string, response http.ResponseWriter) {
	nf := notFoundData{
		Page: path,
	}
	response.WriteHeader(http.StatusNotFound)
	htmlHeader(response)
	err := notFound.Execute(response, &nf)
	if err != nil {
		log.Printf("Error processing not found template: %v\n", err)
		return
	}
}

func (w *webApp) sendJson(value any, errorString string, response http.ResponseWriter) {
	bts, err := json.Marshal(value)
	if err != nil {
		log.Printf(errorString, err)
		w.errorPage("ERROR", renderError(err), http.StatusInternalServerError, response)
		return
	}
	jsonHeader("", response)
	response.WriteHeader(http.StatusOK)
	_, err = response.Write(bts)
	if err != nil {
		log.Printf("Error writing response: %v\n", err)
	}
}

func (w *webApp) sendErrorJson(value any, errorString string, response http.ResponseWriter) {
	bts, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		log.Printf(errorString, err)
		w.errorPage("ERROR", renderError(err), http.StatusInternalServerError, response)
		return
	}
	w.errorPage("ERROR", renderStringError(string(bts)), http.StatusInternalServerError, response)
}

func (w *webApp) sendStatusJson(value any, status int, errorString string, response http.ResponseWriter) {
	bts, err := json.Marshal(value)
	if err != nil {
		log.Printf(errorString, err)
		w.errorPage("ERROR", renderError(err), http.StatusInternalServerError, response)
		return
	}
	jsonHeader("", response)
	response.WriteHeader(status)
	_, err = response.Write(bts)
	if err != nil {
		log.Printf("Error writing response: %v\n", err)
	}
}

func (w *webApp) renderIndex(response http.ResponseWriter, request *http.Request) {
	htmlHeader(response)
	id := wrapperData{
		Title: "Home",
		Body:  templateToHTML(home, nil),
	}
	err := wrapper.Execute(response, &id)
	if err != nil {
		log.Printf("Error generating page: %v\n", err)
		w.errorPage("ERROR", renderError(err), http.StatusInternalServerError, response)
	}
}
