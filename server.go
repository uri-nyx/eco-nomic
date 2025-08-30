package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"time"
	"os"
	"errors"

	"github.com/flytam/filenamify"
	"github.com/google/uuid"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

const TITLE_PROVISIONAL = "NOMIC BANK"

const COOKIE_NAME = "session_token"

var sessions = map[string]session{}

type session struct {
	id     uint64
	holder string
	expiry time.Time
}

type PageData struct {
	Account *Account
	Clock   uint64
	Errors  []string
	Book    []Book
	Lang	string
	Title	string
}

func (s session) isExpired() bool {
	return s.expiry.Before(time.Now())
}


func checkSessionCookie(b *Bank, r *http.Request) (*Account, error) {
	c, err := r.Cookie(COOKIE_NAME)
	if err != nil {
		return nil, err
	}

	sessionToken := c.Value

	userSession, exists := sessions[sessionToken]
	if !exists {
		// If the session token is not present in session map, return an unauthorized error
		return nil, fmt.Errorf("Unauthorized")
	}

	// If the session is present, but has expired, we can delete the session, and return
	// an unauthorized status
	if userSession.isExpired() {
		delete(sessions, sessionToken)
		return nil, fmt.Errorf("Expired")
	}

	a, err := b.LoadAccountById(int64(userSession.id))
	if err != nil {
		// Account not found!
		return nil, fmt.Errorf("No such Account")

	}

	return a, nil
}

func indexHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {
	renderTemplate(w, "index", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: nil, Errors: nil})
}

func loginHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {

	holder := r.FormValue("name")
	password := r.FormValue("password")
	id, err := strconv.ParseUint(r.FormValue("account"), 10, 64)

	var errors []string

	if err != nil {
		// Account number not well formatted
		errors = append(errors, ErrorStrings[lang][ERR_ACCOUNT_NUMBER_INVALID])
	}

	_, err = b.GetAccountHolder(int64(id))
	if err != nil {
		// Account not found i db
		errors = append(errors, ErrorStrings[lang][ERR_ACCOUNT_NOT_FOUND])
	}

	hash, err := b.GetHash(int64(id))
	if err != nil {
		// Account not found i db
		errors = append(errors, ErrorStrings[lang][ERR_ACCOUNT_NOT_FOUND])
	}

	if !CheckPassword(password, hash) {
		// Password incorrect
		errors = append(errors, ErrorStrings[lang][ERR_INCORRECT_PASSWORD])
	}

	if len(errors) > 0 {
		renderTemplate(w, "index", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: nil, Errors: errors})
		return
	}

	sessionToken := uuid.NewString()
	expiresAt := time.Now().Add(360 * time.Second)

	sessions[sessionToken] = session{
		id:     id,
		holder: holder,
		expiry: expiresAt,
	}

	http.SetCookie(w, &http.Cookie{
		Name:    COOKIE_NAME,
		Value:   sessionToken,
		Expires: expiresAt,
		Path:    "/",
	})

	http.Redirect(w, r, "/a/" + lang + "/account/", http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {
	c, err := r.Cookie(COOKIE_NAME)
	if err != nil {
		if err == http.ErrNoCookie {
			// If the cookie is not set, return an unauthorized status
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// For any other type of error, return a bad request status
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sessionToken := c.Value

	delete(sessions, sessionToken)

	// We need to let the client know that the cookie is expired
	// In the response, we set the session token to an empty
	// value and set its expiry as the current time
	http.SetCookie(w, &http.Cookie{
		Name:    COOKIE_NAME,
		Value:   "",
		Expires: time.Now(),
	})

	http.Redirect(w, r, "/a/" + lang, http.StatusFound)
}

func accountHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {
	c, err := r.Cookie(COOKIE_NAME)
	if err != nil {
		if err == http.ErrNoCookie {
			// If the cookie is not set, return an unauthorized status
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// For any other type of error, return a bad request status
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sessionToken := c.Value

	userSession, exists := sessions[sessionToken]
	if !exists {
		// If the session token is not present in session map, return an unauthorized error
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// If the session is present, but has expired, we can delete the session, and return
	// an unauthorized status
	if userSession.isExpired() {
		delete(sessions, sessionToken)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	a, err := b.LoadAccountById(int64(userSession.id))
	if err != nil {
		// Here I think unauthoized is the best response
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	book, err := b.GetBook()
	for i := 0; i < len(book); i++ {
		if book[i].Id == a.Id {
			book = append(book[:i], book[i+1:]...)
			break
		}
	}

	renderTemplate(w, "account", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Account: a, Clock: b.clock, Errors: nil, Book: book})
}

func transferHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {

	var errors []string

	a, err := checkSessionCookie(b, r)
	if err != nil {
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	creditor, err := strconv.ParseUint(r.FormValue("to"), 10, 64)
	if err != nil {
		errors = append(errors, ErrorStrings[lang][ERR_ACCOUNT_NUMBER_INVALID])
	}

	amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	if err != nil {
		errors = append(errors, ErrorStrings[lang][ERR_TRANSFER_AMOUNT_INVALID])
	}

	due, err := strconv.ParseUint(r.FormValue("due"), 10, 64)
	if err != nil {
		errors = append(errors, ErrorStrings[lang][ERR_TRANSFER_DATE_INVALID])
	}

	if len(errors) > 0 {
		renderTemplate(w, "account", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: errors})
		return
	}

	concept := r.FormValue("concept")

	err = b.Transfer(uint64(a.Id), creditor, amount, due, concept)
	if err != nil {
		errors = append(errors, GetBackendError(lang, err.Error()))
		renderTemplate(w, "account", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: errors})
		return
	}

	log.Printf("Transfer Ordered from %s (%d) to %d due on %d for $%d\n", a.Holder, a.Id, creditor, due, amount)
	http.Redirect(w, r, "/a/" + lang + "/account/", http.StatusFound) // maybe some hash encoding or something
}

func changepasswdHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {

	var errors []string

	a, err := checkSessionCookie(b, r)
	if err != nil {
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	currpass := r.FormValue("curr")
	newpass := r.FormValue("new")
	confirmpass := r.FormValue("confirm")

	if newpass != confirmpass {
		errors = append(errors, ErrorStrings[lang][ERR_NEW_PASSWORDS_MISMATCH])
	}

	hash, err := b.GetHash(a.Id)
	if err != nil {
		// Account not found i db
		errors = append(errors, ErrorStrings[lang][ERR_ACCOUNT_NOT_FOUND])
	}

	if !CheckPassword(currpass, hash) {
		// Password incorrect
		errors = append(errors, ErrorStrings[lang][ERR_CURRENT_PASSWORD_INCORRECT])
	}

	if len(errors) > 0 {
		renderTemplate(w, "account", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: errors})
		return
	}

	err = b.ChangePass(a.Id, newpass)
	if err != nil {
		errors = append(errors, GetBackendError(lang, err.Error()))
		renderTemplate(w, "account", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: errors})
		return
	}

	http.Redirect(w, r, "/a/" + lang +"/account/", http.StatusFound) // maybe some hash encoding or something
}

func revokeHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {

	var errors []string
	
	a, err := checkSessionCookie(b, r)
	if err != nil {
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return

	}

	transaction_id, err := strconv.ParseUint(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		errors = append(errors, ErrorStrings[lang][ERR_TRANSACTION_ID_INVALID])
		renderTemplate(w, "account", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: errors})
		return
	}

	err = b.RevokeTransaction(a.Id, transaction_id)
	if err != nil {
		errors = append(errors, GetBackendError(lang, err.Error()))
		renderTemplate(w, "account", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: errors})
		return
	}

	log.Printf("%s (%d) revoked transaction #%d\n", a.Holder, a.Id, transaction_id)
	http.Redirect(w, r, "/a/" + lang +"/account/", http.StatusFound) // maybe some hash encoding or something
}

func readHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {

	var errors []string

	a, err := checkSessionCookie(b, r)
	if err != nil {
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return

	}


	letter_id, err := strconv.ParseUint(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		errors = append(errors, ErrorStrings[lang][ERR_LETTER_ID_INVALID])
		renderTemplate(w, "account", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: errors})
		return
	}

	l, err := a.LoadLetter(letter_id, b)
	if err != nil {
		errors = append(errors, GetBackendError(lang, err.Error()))
		renderTemplate(w, "account", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: errors})
		return
	}

	err = templates.ExecuteTemplate(w, "read.html", &struct{Lang string; ReadLetter Letter}{Lang: lang, ReadLetter: l})
	if err != nil {
		http.Error(w, GetBackendError(lang, err.Error()), http.StatusInternalServerError)
	}
}

func letterHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {
	
	a, err := checkSessionCookie(b, r)
	if err != nil {
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return

	}

	book, err := b.GetBook()
	for i := 0; i < len(book); i++ {
		if book[i].Id == a.Id {
			book = append(book[:i], book[i+1:]...)
			break
		}
	}

	renderTemplate(w, "letter", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: nil, Book: book})
}

func bookHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {
	
	_, err := checkSessionCookie(b, r)
	if err != nil {
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return

	}

	book, err := b.GetBook()
	if err != nil {
		http.Error(w, GetBackendError(lang, err.Error()), http.StatusInternalServerError)
	}

	err = templates.ExecuteTemplate(w, "book.html", &struct{Lang string; AccountsBook []Book}{Lang: lang, AccountsBook:book})
	if err != nil {
		http.Error(w, GetBackendError(lang, err.Error()), http.StatusInternalServerError)
	}
}

func archiveHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {

	archive, err := b.GetArchive()
	if err != nil {
		http.Error(w, GetBackendError(lang, err.Error()), http.StatusInternalServerError)
	}

	err = templates.ExecuteTemplate(w, "archive.html", &struct{Lang string; PublicArchive []Letter}{Lang: lang, PublicArchive: archive})
	if err != nil {
		http.Error(w, GetBackendError(lang, err.Error()), http.StatusInternalServerError)
	}
}

func docHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {

	doc_id, err := strconv.ParseUint(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		http.Error(w, GetBackendError(lang, err.Error()), http.StatusInternalServerError)
		return
	}

	var l Letter
	l, err = b.LoadDoc(doc_id)
	if err != nil {
		http.Error(w, GetBackendError(lang, err.Error()), http.StatusInternalServerError)
	}

	err = templates.ExecuteTemplate(w, "read.html", &struct{Lang string; ReadLetter Letter}{Lang: lang, ReadLetter: l})
	if err != nil {
		http.Error(w, GetBackendError(lang, err.Error()), http.StatusInternalServerError)
	}
}


func sendHandler(w http.ResponseWriter, r *http.Request, b *Bank, lang string) {
		
	a, err := checkSessionCookie(b, r)
	if err != nil {
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return

	}

	receiver, err := strconv.ParseInt(r.FormValue("to"), 10, 64)
	if err != nil {
		renderTemplate(w, "letter", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: []string{GetBackendError(lang, err.Error())}})
	}

	body := r.FormValue("body")
	title := r.FormValue("title")

	now := time.Now()
	time := now.Unix()
	directory := fmt.Sprintf("%d-%s", a.Id, a.Holder)
	clock := fmt.Sprintf("%d", b.clock)
	name, err := filenamify.Filenamify(title, filenamify.Options{
		Replacement: "_",
	})

	if err != nil {
		http.Error(w, GetBackendError(lang, err.Error()), http.StatusInternalServerError)
		return
	}

	path := path.Join("bank", "letters", directory, clock+"-"+now.Format("2006_01_02")+"_"+name+"_to_"+r.FormValue("to")+".txt")

	l := &Letter{Timestamp: uint64(time), Sender: a.Id, Receiver: receiver, Date: b.clock, Path: path, Title: title, Body: []byte(body)}

	if r.FormValue("send") != "" {
		err = l.Send(b)
	} else if r.FormValue("publish") != "" {
		err = l.Publish(b)
	} else {

	}

	if err != nil {
		renderTemplate(w, "letter", &PageData{Title: TITLE_PROVISIONAL, Lang: lang, Clock: b.clock, Account: a, Errors: []string{GetBackendError(lang, err.Error())}})
		return
	}

	log.Println("Received letter from " + a.Holder + " at date " + string(b.clock) + ": " + title)
	http.Redirect(w, r, "/a/" + lang +"/account", http.StatusFound)
}

var templates = template.Must(template.ParseFiles("tmpl/index.html", "tmpl/account.html", "tmpl/letter.html", "tmpl/read.html", "tmpl/book.html", "tmpl/archive.html"))

func renderTemplate(w http.ResponseWriter, tmpl string, d *PageData) {
	err := templates.ExecuteTemplate(w, tmpl+".html", d)
	if err != nil {
		http.Error(w,  err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/a/(es|en)/(account/|archive/|transfer/|login/|send/|letter/|logout/|book/|changepasswd/|revoke/[0-9]+|read/[0-9]+|doc/[0-9]+)?$")

func makeHandler(fn func(http.ResponseWriter, *http.Request, *Bank, string), b *Bank) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !validPath.MatchString(r.URL.Path) {
			log.Println("Path not valid")
			http.NotFound(w, r)
			return
		}
		lang := r.PathValue("lang")
		fn(w, r, b, lang)
	}
}

func main() {

	args := os.Args
	if len(args) != 2 {
		fmt.Printf("Usage: %s <db-filename>\n", args[0])
		os.Exit(1)
	}

	dbfname := args[1]
	if _, err := os.Stat(dbfname); errors.Is(err, os.ErrNotExist) {
		fmt.Println("No such database file: ", dbfname)
		os.Exit(1)
	}


	bank, err := OpenBank(dbfname)

	if err != nil {
		log.Fatal(err)
	}

	
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	
	http.HandleFunc("/a/{lang}/", makeHandler(indexHandler, bank))
	http.HandleFunc("/a/{lang}/login/", makeHandler(loginHandler, bank))
	http.HandleFunc("/a/{lang}/logout/", makeHandler(logoutHandler, bank))
	http.HandleFunc("/a/{lang}/account/", makeHandler(accountHandler, bank))
	http.HandleFunc("/a/{lang}/transfer/", makeHandler(transferHandler, bank))
	http.HandleFunc("/a/{lang}/revoke/", makeHandler(revokeHandler, bank))
	http.HandleFunc("/a/{lang}/letter/", makeHandler(letterHandler, bank))
	http.HandleFunc("/a/{lang}/send/", makeHandler(sendHandler, bank))
	http.HandleFunc("/a/{lang}/read/", makeHandler(readHandler, bank))
	http.HandleFunc("/a/{lang}/book/", makeHandler(bookHandler, bank))
	http.HandleFunc("/a/{lang}/archive/", makeHandler(archiveHandler, bank))
	http.HandleFunc("/a/{lang}/doc/", makeHandler(docHandler, bank))
	http.HandleFunc("/a/{lang}/changepasswd/", makeHandler(changepasswdHandler, bank))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
