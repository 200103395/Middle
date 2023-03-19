package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"golang.org/x/crypto/bcrypt"
	"html/template"
	"net/http"
	"strings"
	"unicode"
)

var tpl *template.Template
var db *sql.DB
var store = sessions.NewCookieStore([]byte("super-secret"))

type Book struct {
	Name   string
	Author string
	Price  string
}

type SentTemp struct {
	Username string
	Books    []Book
}
type Nofound struct {
	Error   string
	Checker bool
}

var SentTemplate SentTemp

func main() {
	var err error
	tpl, err = tpl.ParseGlob("templates/*.html")

	if err != nil {
		fmt.Println("Template parsing error")
		panic(err.Error())
	}

	db, err = sql.Open("mysql", "root:Ak200222!@tcp(localhost:3306)/testdb")
	if err != nil {
		fmt.Println("Error connecting to database")
		panic(err.Error())
	}
	var books *[]Book
	books = &SentTemplate.Books
	rows, _ := db.Query("SELECT * from Books")
	for rows.Next() {
		var name string
		var price string
		var author string
		var id int
		err := rows.Scan(&id, &name, &price, &author)
		if err != nil {
			panic(err)
		}
		book := Book{name, author, price}
		*books = append(*books, book)
	}
	defer db.Close()
	http.HandleFunc("/register", RegisterHandler)
	http.HandleFunc("/registerconfirm", RegisterConfirmationHandler)
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/loginconfirm", LoginConfirmationHadler)
	http.HandleFunc("/", MainPageHandler)
	http.HandleFunc("/logout", LogoutHandler)
	http.HandleFunc("/book", BookHandler)
	http.ListenAndServe("localhost:8080", nil)
}

func Cookie(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	username, ok := session.Values["username"]
	fmt.Println("ok:", ok)
	if !ok {
		SentTemplate.Username = ""
		return
	}
	SentTemplate.Username = fmt.Sprint(username)
}

func BookHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	searchName := r.FormValue("name")
	fmt.Println(searchName)
	var Books []Book
	rows, _ := db.Query("SELECT * from Books")
	for rows.Next() {
		var name string
		var price string
		var author string
		var id int
		err := rows.Scan(&id, &name, &price, &author)
		if err != nil {
			panic(err)
		}
		//Compare book with searched book
		if strings.Contains(strings.ToLower(name), strings.ToLower(searchName)) {
			book := Book{name, author, price}
			Books = append(Books, book)
		}
	}
	fmt.Println(SentTemplate)
	Cookie(w, r)
	SentTe := SentTemp{SentTemplate.Username, Books}
	if len(Books) == 0 {
		tpl.ExecuteTemplate(w, "book.html", Nofound{"No book found", true})
	} else {
		tpl.ExecuteTemplate(w, "book.html", SentTe)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Main page running")
	Cookie(w, r)
	tpl.ExecuteTemplate(w, "mainpage.html", SentTemplate)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Login handler successfully running")
	tpl.ExecuteTemplate(w, "login.html", nil)
}

func LoginConfirmationHadler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Login confirmation successfully running")
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")
	var hash string
	statement := "SELECT password from Users WHERE username = ?"
	row := db.QueryRow(statement, username)
	err := row.Scan(&hash)
	fmt.Println("hash from db:", hash)
	if err != nil {
		fmt.Println("Error taking hash from db")
		tpl.ExecuteTemplate(w, "login.html", "Check username and password")
		return
	}
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		fmt.Println("Login successfully")
		session, _ := store.Get(r, "session")
		session.Values["username"] = username
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusFound)
		//tpl.ExecuteTemplate(w, "mainpage.html", username)
		return
	}
	fmt.Println("Incorrect password")
	tpl.ExecuteTemplate(w, "login.html", "check username and password")
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Register handler successfully running")
	tpl.ExecuteTemplate(w, "register.html", nil)
}

func RegisterConfirmationHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Register confirmation handler successfully running")
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")
	var UsernameValid = true
	for _, char := range username {
		if unicode.IsDigit(char) == false && unicode.IsLetter(char) == false {
			UsernameValid = false
		}
	}
	if UsernameValid == false || len(username) > 40 || len(username) < 5 {
		tpl.ExecuteTemplate(w, "register.html", "Invalid username")
		return
	}
	fmt.Println("Parsing successfull")
	var Uppercase, Lowercase, Digit, Noother bool
	Noother = true
	for _, char := range password {
		if unicode.IsDigit(char) {
			Digit = true
		}
		if unicode.IsLower(char) {
			Lowercase = true
		}
		if unicode.IsUpper(char) {
			Uppercase = true
		}
		if unicode.IsDigit(char) == false && unicode.IsLetter(char) == false {
			Noother = false
		}
	}
	if !Uppercase || !Lowercase || !Digit || !Noother || len(password) < 8 || len(password) > 20 {
		tpl.ExecuteTemplate(w, "register.html", "Invalid password")
		return
	}
	statement := "SELECT UserID from Users WHERE username = ?"
	row := db.QueryRow(statement, username)
	var UserID int
	err := row.Scan(&UserID)
	fmt.Println(UserID)
	if err != sql.ErrNoRows {
		tpl.ExecuteTemplate(w, "register.html", "Username already taken")
		return
	}
	fmt.Println("Username unique")

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	fmt.Println(hash)
	if err != nil {
		tpl.ExecuteTemplate(w, "register.html", "Error hashing password")
		return
	}
	insert, err := db.Prepare("INSERT INTO Users (Username, Password) Values(?,?)")
	if err != nil {
		tpl.ExecuteTemplate(w, "register.html", "Error inserting data")
		return
	}
	result, err := insert.Exec(username, hash)
	if err != nil {
		panic(err.Error())
		tpl.ExecuteTemplate(w, "register.html", "Error inserting data")
		return
	}
	fmt.Println(result.RowsAffected())
	fmt.Println("User created successfully")
	http.Redirect(w, r, "/", http.StatusFound)
	//tpl.ExecuteTemplate(w, "mainpage.html", username)
}
