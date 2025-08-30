package main

import (
	"log"
	"regexp"
	"strconv"
	"database/sql"
	"fmt"
	"path"
	"os"
	"bytes"
	"html/template"


	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/extension"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

type Account struct {
	Id int64
	Holder string
	Date uint64
	Balance int64
	Transactions []Transaction
	Letters []Letter
}

type Transaction struct {
	Id int64
	Date uint64
	Concept string
	Amount int64
	Creditor int64
	Debitor int64
	Payed bool
	Revoked bool
	To_from string
}

type Letter struct {
	Timestamp uint64
	Sender int64
	Receiver int64
	Path string
	Title string
	Body []byte
	Html template.HTML
	Date uint64
	From string
	To string
	Public bool
}

type Bank struct {
	db *sql.DB
	clock uint64
}

var md goldmark.Markdown =  goldmark.New(
	goldmark.WithExtensions(extension.GFM, extension.DefinitionList, extension.Typographer),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()))

func (b *Bank) LoadDoc(letter_id uint64) (Letter, error) {
	// Load document and its metadata (the same as a Letter)
	var l Letter
	err := b.db.QueryRow(
		"SELECT sender, Path, Title, date, coalesce(public, 0), id FROM letters WHERE id = $1 and public = 1 ORDER BY id ASC;",
		letter_id).Scan(&l.Sender, &l.Path, &l.Title, &l.Date, &l.Public, &l.Timestamp)


	if err != nil {
		return l, fmt.Errorf(ERR_DOC_NOT_FOUND)
	}

	log.Println("Reading file ", l.Path)

	txt, err := os.ReadFile(l.Path)
	if err != nil {
		return l, err
	}

	l.Body = txt

	var buf bytes.Buffer
	err = md.Convert(txt, &buf)
	if err != nil {
		return l, err
	}

	l.Html = template.HTML(buf.String())

	l.From, err = b.GetAccountHolder(l.Sender)
	if err != nil {
		return l, err
	}

	return l, nil
}

func (a *Account) LoadLetter(letter_id uint64, b *Bank) (Letter, error) {
	// check if account is either sender or receiver of this letter
	var l Letter
	found := false
	for i:=0; i<len(a.Letters);i++ {
		if a.Letters[i].Timestamp == letter_id {
			l = a.Letters[i]
			found = true
			break
		}
	}

	if !found {
		return l, fmt.Errorf(ERR_LETTER_NOT_IN_INBOX)
	}

	txt, err := os.ReadFile(l.Path)
	if err != nil {
		return l, err
	}

	l.Body = txt

	var buf bytes.Buffer
	err = md.Convert(txt, &buf)
	if err != nil {
		return l, err
	}

	l.Html = template.HTML(buf.String())

	l.From, err = b.GetAccountHolder(l.Sender)
	if err != nil {
		return l, err
	}

	l.To, err = b.GetAccountHolder(l.Receiver)
	if err != nil {
		return l, err
	}

	return l, nil
}


func (l *Letter) Send(b *Bank) error {
	insert := `
	INSERT INTO letters 
	(id, sender, receiver, Title, Path, Date, public)
	VALUES
	($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := b.GetAccountHolder(l.Receiver) 
	if err != nil {
		return err
	}
	
	_, err = b.db.Exec(insert, l.Timestamp, l.Sender, l.Receiver, l.Title, l.Path, l.Date, false)
	if err != nil {
		log.Println("Error inserting: " + err.Error())
		return err
	}

	err = os.MkdirAll(path.Dir(l.Path), 0600)

	if err != nil {
		log.Println("Error creatinG directories: " + err.Error())
		return err
	}

	return os.WriteFile(l.Path, l.Body, 0600)
}

func (l *Letter) Publish(b *Bank) error {
	insert := `
	INSERT INTO letters 
	(id, sender, receiver, Title, Path, Date, public)
	VALUES
	($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := b.GetAccountHolder(l.Receiver) 
	if err != nil {
		return err
	}
	
	_, err = b.db.Exec(insert, l.Timestamp, l.Sender, l.Receiver, l.Title, l.Path, l.Date, true)
	if err != nil {
		log.Println("Error inserting: " + err.Error())
		return err
	}

	err = os.MkdirAll(path.Dir(l.Path), 0600)

	if err != nil {
		log.Println("Error creatin directories: " + err.Error())
		return err
	}

	err = os.MkdirAll("static/archive", 0600)

	err = os.WriteFile(l.Path, l.Body, 0600)
	if err != nil {
		log.Println("Error writing file: " + err.Error())
		return err
	}

	return os.WriteFile(path.Join("static/", "archive", path.Base(l.Path)), l.Body, 0600)
}


func OpenBank(filename string) (*Bank, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	
	var clock uint64
	err = db.QueryRow("SELECT clock FROM system WHERE id = 1;").Scan(&clock)
	if err != nil {
		return nil, err
	}

	return &Bank{db: db, clock: clock}, nil
}

func (b *Bank) Close() error {
	return b.db.Close()
}

func (b *Bank) GetDate() uint64 {
	err := b.db.QueryRow("SELECT clock FROM system WHERE id = 1;").Scan(&b.clock)
	if err != nil {
		log.Fatal("Error querying: " + err.Error())
	}

	return b.clock
}

type Book struct {
	Holder string
	Id int64
}

func (b *Bank) GetBook() ([]Book, error){
	rows, err := b.db.Query("SELECT id, holder FROM accounts;")

	if err != nil {
		return nil, err
	}

	var book []Book


	for rows.Next() {
		var bo Book
		if err := rows.Scan(&bo.Id, &bo.Holder); err != nil {
			return nil, err
		}

		if bo.Id >= 0 {
			book = append(book, bo)
		}
		
	}

	return book, nil
}


func (b *Bank) GetArchive() ([]Letter, error){
	rows, err := b.db.Query("SELECT id, sender, receiver, Title, Path, Date FROM letters WHERE public = 1;")

	if err != nil {
		return nil, err
	}

	var archive []Letter


	for rows.Next() {
		var l Letter
		if err := rows.Scan(&l.Timestamp, &l.Sender, &l.Receiver, &l.Title, &l.Path, &l.Date); err != nil {
			return nil, err
		}

		l.Path = path.Join("static", "archive", path.Base(l.Path))
		l.From, err = b.GetAccountHolder(l.Sender)

		if err  != nil {
			return nil, err
		}

		archive = append(archive, l)
	}

	return archive, nil
}


func (b *Bank) GetAccountHolder(id int64) (string, error) {
	var holder string
	err := b.db.QueryRow("SELECT holder FROM accounts WHERE id = $1;", id).Scan(&holder)
	if err != nil {
		return "", err
	}

	return holder, nil
}

func (b *Bank) GetHash(id int64) ([]byte, error) {
	var password []byte
	err := b.db.QueryRow("SELECT password FROM accounts WHERE id = $1;", id).Scan(&password)
	if err != nil {
		return []byte("none"), err
	}

	return password, nil
}

func (b *Bank) ChangePass(id int64, pass string) error {
	hash, err := CreateHash(pass)

	if err != nil {
		return err
	}

	_, err = b.db.Exec("UPDATE accounts SET password = $1 WHERE id = $2", hash, id)

	return err
}

func (b *Bank) LoadAccountById(id int64) (*Account, error) {
	var a Account
	err := b.db.QueryRow("SELECT id, holder, date FROM accounts WHERE id = $1;", id).Scan(&a.Id, &a.Holder, &a.Date)
	if err != nil {
		return nil, err
	}

	a.Balance = b.balance(id)
	a.Transactions, err = b.getTransactions(id)
	if err != nil {
		log.Println("Error querying: " + err.Error())
		return nil, err
	}

	a.Letters, err = b.getLetters(id)
	if err != nil {
		log.Println("Error querying: " + err.Error())
		return nil, err
	}

	return &a, nil
}


var validCredentials = regexp.MustCompile("^([a-zA-Z]+)-([0-9]+)$")
func (b *Bank) LoadAccount(credentials string) (*Account) {
	// tidy uo this
	m := validCredentials.FindStringSubmatch(credentials)
	if m == nil {
		log.Println("Invalid credentials")
		return nil
	}

	holder := m[1]
	id, err := strconv.ParseUint(m[2], 10, 64)
	if err != nil {
		log.Println("Error converting: " + err.Error())
		return nil
	}

	var date uint64
	err = b.db.QueryRow("SELECT date FROM accounts WHERE id = $1;", id).Scan(&date)
	if err != nil {
		log.Println("Error querying: " + err.Error())
		return nil
	}


	balance := b.balance(int64(id))

	transactions, err := b.getTransactions(int64(id))
	
	if err != nil {
		log.Println("Error querying: " + err.Error())
		return nil
	}
	
	letters, err := b.getLetters(int64(id))
	if err != nil {
		log.Println("Error querying: " + err.Error())
		return nil
	}

	return &Account{Id: int64(id), Holder: holder, Date: date, Balance: balance, Transactions: transactions, Letters: letters}
} 

func (b *Bank) Transfer(from uint64, to uint64, amount int64, due uint64, concept string) error {
	insert := `
		INSERT INTO transactions 
		(creditor, debitor, amount, concept, date_created, date_due, payed, revoked)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8)
	`

	// TODO: check if transaction is valid, and tidy up error messages
	// cannot transfer to self!
	if from == to {
		return fmt.Errorf(ERR_TRANSFER_TO_SELF)
	}

	// cannot transfer if balance is lesser than amount
	if amount > b.balance(int64(from)) {
		return fmt.Errorf(ERR_INSUFFICIENT_FUNDS)
	}

	// cannot transfer negative moneys
	if amount < 0 {
		return fmt.Errorf(ERR_NEGATIVE_TRANSFER_AMOUNT)
	}

	// cannot transfer in the past!
	if due < b.clock {
		return fmt.Errorf(ERR_TIME_TRAVEL_IMPOSSIBLE)
	}

	// cannot transfer to no one or a non existing account
	_, err := b.GetAccountHolder(int64(to))
	if err != nil {
		return fmt.Errorf(ERR_RECIPIENT_ACCOUNT_NOT_FOUND)
	}


	payed := due == b.GetDate() 

	// Sanitize input and checks put in banking
	_, err = b.db.Exec(insert, to, from, amount, concept, b.clock, due, payed, false)
	if err != nil {
		log.Println("Error inserting: " + err.Error())
		return err
	}
	return nil
}

func (b *Bank) balance(id int64) int64 {
	// error handling
	debits_query := "select coalesce(sum(amount), 0) from transactions where debitor = $1 and payed = 1 and revoked = 0;"
	credits_query := "select coalesce(sum(amount), 0) from transactions where creditor = $1 and payed = 1 and revoked = 0;"
	
	_ = b.GetDate()
	
	var credits, debits int64
	err := b.db.QueryRow(debits_query, id).Scan(&debits)
	if err != nil {
		log.Println("Error querying for debits: " + err.Error())
		return 0
	}
	err = b.db.QueryRow(credits_query, id).Scan(&credits)
	if err != nil {
		log.Println("Error querying for credits: " + err.Error())
		return 0
	}
	return credits - debits
}



func (b *Bank) getTransactions(id int64) ([]Transaction, error) {
	rows, err := b.db.Query("SELECT id, date_due, concept, amount, creditor, debitor, payed FROM transactions WHERE (debitor = $1 or creditor = $2) and revoked = 0 ORDER BY id DESC;", id, id)

	if err != nil {
		return nil, err
	}

	transactions := []Transaction{}

	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.Id, &t.Date, &t.Concept, &t.Amount, &t.Creditor, &t.Debitor, &t.Payed); err != nil {
			return nil, err
		}
		
		transactions = append(transactions, t)
	}

	/*
	ID, Date, Due, Concept, Amount, To/From

	*/

	for i := 0; i < len(transactions); i++ {
		if transactions[i].Creditor == id {
			debitor, err := b.GetAccountHolder(transactions[i].Debitor)
			if err != nil {
				return nil, err
			}

			if transactions[i].Debitor < 0 { 
				transactions[i].To_from = fmt.Sprintf("<-- %s", debitor) 
			} else { 
				transactions[i].To_from = fmt.Sprintf("<-- %s [%04d]", debitor, transactions[i].Debitor) 
			}
		} else {
			creditor, err := b.GetAccountHolder(transactions[i].Creditor)
			if err != nil {
				return nil, err
			}

			if transactions[i].Creditor < 0 {
				transactions[i].To_from = fmt.Sprintf("--> %s", creditor)
			} else { 
				transactions[i].To_from = fmt.Sprintf("--> %s [%04d]", creditor, transactions[i].Creditor)
			}

			transactions[i].Amount = - transactions[i].Amount
		}
	}

	return transactions, nil
}


func (b *Bank) getLetters(id int64) ([]Letter, error) {
	rows, err := b.db.Query("SELECT id, sender, receiver, Title, Path, date, coalesce(public, 0) FROM letters WHERE sender = $1 or receiver = $2 or public = 1 ORDER BY id ASC;", id, id)

	if err != nil {
		return nil, err
	}

	letters := []Letter{}

	for rows.Next() {
		var l Letter
		if err := rows.Scan(&l.Timestamp, &l.Sender, &l.Receiver, &l.Title, &l.Path, &l.Date, &l.Public); err != nil {
			return nil, err
		}
		
		letters = append(letters, l)
	}

	/*
	ID, Date, Due, Concept, Amount, To/From

	*/

	for i := 0; i < len(letters); i++ {
		if letters[i].Sender == id {
			letters[i].To, err = b.GetAccountHolder(letters[i].Receiver)
			if err  != nil {
				return nil, err
			}
			letters[i].From = "-"

		} else {
			letters[i].From, err = b.GetAccountHolder(letters[i].Sender)
			if err  != nil {
				return nil, err
			}
			letters[i].To = "-"
		}
	}

	return letters, nil
}

func (b *Bank) getTransaction(id uint64) (Transaction, error) {
	var t Transaction
	err := b.db.QueryRow("SELECT id, date_due, concept, amount, creditor, debitor, payed FROM transactions WHERE id = $1;", id).Scan(&t.Id, &t.Date, &t.Concept, &t.Amount, &t.Creditor, &t.Debitor, &t.Payed)

	if err != nil {
		return t, err
	}

	return t, nil
}

func (b *Bank) getLetter(id uint64) (Letter, error) {
	var l Letter
	err := b.db.QueryRow("SELECT id,sender,receiver,Title,Path,Date FROM letters WHERE id = $1;", id).Scan(&l.Timestamp, &l.Sender, &l.Receiver, &l.Title, &l.Path, &l.Date)

	if err != nil {
		return l, err
	}

	return l, nil
}


func (b *Bank) RevokeTransaction(account_id int64, transaction_id uint64) error {

	t, err := b.getTransaction(transaction_id)

	if err != nil {
		return err
	}

	_ = b.GetDate()

	if (t.Creditor == account_id || t.Debitor == account_id) && !t.Payed && (t.Date > b.clock) {
		_, err = b.db.Exec("UPDATE transactions SET revoked = $1 WHERE id = $2;", true, t.Id)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf(ERR_RECIPIENT_ACCOUNT_NOT_FOUND)
	}

	return nil
}