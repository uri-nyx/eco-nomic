// Copyright 2010 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore

package main

import (
	"html/template"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
	"path"
	"fmt"

	"github.com/google/uuid"
	"github.com/flytam/filenamify"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

const COOKIE_NAME = "session_token"
var sessions = map[string]session{}

type session struct {
	id uint64
	holder string
	expiry time.Time
}

type PageData struct {
	Account *Account
	Clock uint64
	Errors []string
}

func (s session) isExpired() bool {
	return s.expiry.Before(time.Now())
}

func indexHandler(w http.ResponseWriter, r *http.Request, b *Bank) {
	renderTemplate(w, "index", &PageData{Clock: b.clock, Account: nil, Errors: nil})
}

func loginHandler(w http.ResponseWriter, r *http.Request, b *Bank) {
	
	holder := r.FormValue("name")
	//password := r.FormValue("password")
	id, err := strconv.ParseUint(r.FormValue("account"), 10, 64)

	var errors []string

	if err != nil {
		// Account number not well formatted
		errors = append(errors, "El número de cuenta debe ser un entero positivo")
	}

	_, err = b.GetAccountHolder(int64(id))
	if err != nil {
		// Account not found i db
		errors = append(errors, "Cuenta no encontrada (debe darse de alta)")
	}

	/*
	if !CheckPassword(holder, id, password) {
		// Password incorrect
		errors = append(errors, "Incorrect password")
	}
	*/

	if len(errors) > 0 {
		renderTemplate(w, "index", &PageData{Clock: b.clock, Account: nil, Errors: errors})
		return
	}

	sessionToken := uuid.NewString()
	expiresAt := time.Now().Add(360 * time.Second)

	sessions[sessionToken] = session{
		id: id,
		holder: holder,
		expiry: expiresAt,
	}

	http.SetCookie(w, &http.Cookie{
		Name:    COOKIE_NAME,
		Value:   sessionToken,
		Expires: expiresAt,
		Path: "/",

	})

	http.Redirect(w, r, "/account/", http.StatusFound)
}

func logoutHandler(w http.ResponseWriter, r *http.Request, b *Bank) {
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

	http.Redirect(w, r, "/", http.StatusFound)
}


func accountHandler(w http.ResponseWriter, r *http.Request, b *Bank) {
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

	a, err:= b.LoadAccountById(int64(userSession.id))
	if err != nil {
		// Here I think unauthoized is the best response
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	renderTemplate(w, "account", &PageData{Account: a, Clock: b.clock, Errors: nil})
}

func transferHandler(w http.ResponseWriter, r *http.Request, b *Bank) {

	// TODO: make a cookie check function
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

	var errors []string

	a, err := b.LoadAccountById(int64(userSession.id))
	if err != nil {
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	creditor, err := strconv.ParseUint(r.FormValue("to"), 10, 64)
	if err != nil {
		errors = append(errors, "El número de cuenta debe ser un entero positivo")
	}

	amount, err := strconv.ParseInt(r.FormValue("amount"), 10, 64)
	if err != nil {
		errors = append(errors, "El importe a transferir debe ser un entero positivo")
	}

	due, err := strconv.ParseUint(r.FormValue("due"), 10, 64)
	if err != nil {
		errors = append(errors, "La fecha debe ser un entero positivo")
	}

	if len(errors) > 0 {
		renderTemplate(w, "account", &PageData{Clock: b.clock, Account: a, Errors: errors})
		return
	}

	concept := r.FormValue("concept")

	
	err = b.Transfer(uint64(a.Id), creditor, amount, due, concept)
	if err != nil {
		errors = append(errors, err.Error())
		renderTemplate(w, "account", &PageData{Clock: b.clock, Account: a, Errors: errors})
		return
	}

	log.Printf("Transfer Ordered from %s (%d) to %d due on %d for $%d\n", a.Holder, a.Id, creditor, due, amount)
	http.Redirect(w, r, "/account/", http.StatusFound) // maybe some hash encoding or something
}

func revokeHandler(w http.ResponseWriter, r *http.Request, b *Bank) {

	// TODO: make a cookie check function
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
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return
	
	}
	var errors []string

	transaction_id, err := strconv.ParseUint(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		errors = append(errors, "Identificador de transacción erróneo")
		renderTemplate(w, "account", &PageData{Clock: b.clock, Account: a, Errors: errors})
		return
	}

	err = b.RevokeTransaction(a.Id, transaction_id)
	if err != nil {
		errors = append(errors, err.Error())
		renderTemplate(w, "account", &PageData{Clock: b.clock, Account: a, Errors: errors})
		return
	}

	log.Printf("%s (%d) revoked transaction #%d\n", a.Holder, a.Id, transaction_id)
	http.Redirect(w, r, "/account/", http.StatusFound) // maybe some hash encoding or something
}


func readHandler(w http.ResponseWriter, r *http.Request, b *Bank) {

	// TODO: make a cookie check function
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
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return
	
	}

	var errors []string

	letter_id, err := strconv.ParseUint(path.Base(r.URL.Path), 10, 64)
	if err != nil {
		errors = append(errors, "Identificador de carta erróneo")
		renderTemplate(w, "account", &PageData{Clock: b.clock, Account: a, Errors: errors})
		return
	}

	l, err := a.LoadLetter(letter_id, b)
	if err != nil {
		errors = append(errors, err.Error())
		renderTemplate(w, "account", &PageData{Clock: b.clock, Account: a, Errors: errors})
		return
	}

	err = templates.ExecuteTemplate(w, "read.html", l)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func letterHandler(w http.ResponseWriter, r *http.Request, b *Bank) {
	// TODO: make a cookie check function
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
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return

	}

	renderTemplate(w, "letter", &PageData{Clock: b.clock, Account: a, Errors: nil})
}


func bookHandler(w http.ResponseWriter, r *http.Request, b *Bank) {
	// TODO: make a cookie check function
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


	_, err = b.LoadAccountById(int64(userSession.id))
	if err != nil {
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return

	}

	book, err :=  b.GetBook()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	
	err = templates.ExecuteTemplate(w, "book.html", book)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func sendHandler(w http.ResponseWriter, r *http.Request, b *Bank) {
	// TODO: make a cookie check function
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
		// Account not found!
		w.WriteHeader(http.StatusUnauthorized)
		return

	}

	receiver, err := strconv.ParseInt(r.FormValue("to"), 10, 64)
	if err != nil {
		renderTemplate(w, "letter", &PageData{Clock: b.clock, Account: a, Errors:[]string{err.Error()}})
	}

	
	body := r.FormValue("body")
	title := r.FormValue("title")

	now := time.Now()
	time := now.Unix()
	directory := fmt.Sprintf("%d-%s", a.Id, a.Holder)
	clock := fmt.Sprintf("%d", b.clock)
	name, err := filenamify.Filenamify(title,filenamify.Options{
    	Replacement:"_",
    })

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	path := path.Join("bank", "letters", directory, clock + "-" + now.Format("2006_01_02") + "_" + name + "_to_" + r.FormValue("to") +  ".txt")

	l := &Letter{Timestamp: uint64(time), Sender: a.Id,  Receiver: receiver, Date: b.clock, Path: path, Title: title, Body: []byte(body)}
	
	err = l.Send(b)
	
	if err != nil {
		renderTemplate(w, "letter", &PageData{Clock: b.clock, Account: a, Errors: []string{err.Error()}})
		return
	}
	
	log.Println("Received letter from " + a.Holder + " at date " + string(b.clock) + ": " + title)
	http.Redirect(w, r, "/account", http.StatusFound)
}


var templates = template.Must(template.ParseFiles("tmpl/index.html", "tmpl/account.html", "tmpl/letter.html", "tmpl/read.html", "tmpl/book.html"))


func renderTemplate(w http.ResponseWriter, tmpl string, d *PageData) {
	err := templates.ExecuteTemplate(w, tmpl+".html", d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var validPath = regexp.MustCompile("^/(account/|transfer/|login/|send/|letter/|logout/|book/|revoke/[0-9]+|read/[0-9]+)?$") // |edit|save|view

func makeHandler(fn func(http.ResponseWriter, *http.Request, *Bank), b *Bank) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ! validPath.MatchString(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		fn(w, r, b)
	}
}


func main() {

	bank, err := OpenBank("file:bank.sqlite3")

	if err != nil {
		log.Fatal(err)
	}


	http.HandleFunc("/", makeHandler(indexHandler, bank))
	http.HandleFunc("/login/", makeHandler(loginHandler, bank))
	http.HandleFunc("/logout/", makeHandler(logoutHandler, bank))
	http.HandleFunc("/account/", makeHandler(accountHandler, bank))
	http.HandleFunc("/transfer/", makeHandler(transferHandler, bank))
	http.HandleFunc("/revoke/", makeHandler(revokeHandler, bank))
	http.HandleFunc("/letter/", makeHandler(letterHandler, bank))
	http.HandleFunc("/send/", makeHandler(sendHandler, bank))
	http.HandleFunc("/read/", makeHandler(readHandler, bank))
	http.HandleFunc("/book/", makeHandler(bookHandler, bank))
	//http.HandleFunc("/codex/", makeHandler(codexHandler, bank)) // An archive of rules and regulations
	//http.HandleFunc("/chronicle/", makeHandler(codexHandler, bank)) // An official log updated each date

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))


/*	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))
*/

	log.Fatal(http.ListenAndServe(":8080", nil))
}
