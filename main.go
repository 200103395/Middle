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

// Struct type that contain all info about book, except id since we don't need it anyways
type Book struct {
	Name   string
	Author string
	Price  string
}

// Struct type that we send to templates, consist of username to show visually that we are logged in and books info from database
type SentTemp struct {
	Username   string
	Books      []Book
	SearchName string
}

var SentTemplate SentTemp

func main() {
	Database()
	var err error
	//Template struct that is used to open and send data in html files
	tpl, err = tpl.ParseGlob("templates/*.html")
	if err != nil {
		fmt.Println("Template parsing error")
		panic(err.Error())
	}

	//defer db.Close()
	//Handle functions that opens related function for every url
	http.HandleFunc("/register", RegisterHandler)
	http.HandleFunc("/registerconfirm", RegisterConfirmationHandler)
	http.HandleFunc("/login", LoginHandler)
	http.HandleFunc("/loginconfirm", LoginConfirmationHadler)
	http.HandleFunc("/", MainPageHandler)
	http.HandleFunc("/logout", LogoutHandler)
	http.HandleFunc("/book", BookHandler)
	http.ListenAndServe("localhost:8080", nil)
}

func Database() {
	//Function that connects to database and takes all information from books table
	var err error
	db, err = sql.Open("mysql", "root:Ak200222!@tcp(localhost:3306)/testdb")
	if err != nil {
		fmt.Println("Error connecting to database")
		panic(err.Error())
	}
	//Taking information only once since we don't change books table in site
	var books *[]Book
	books = &SentTemplate.Books //Changes books variable that are send to templates
	rows, _ := db.Query("SELECT * from Books")
	for rows.Next() {
		var name string
		var price string
		var author string
		var id int
		err = rows.Scan(&id, &name, &price, &author)
		if err != nil {
			panic(err)
		}
		book := Book{name, author, price}
		*books = append(*books, book)
	}
}

func Cookie(r *http.Request) {
	//Function that gets session cookie and prepares it for sending to templates
	session, _ := store.Get(r, "session")
	username, ok := session.Values["username"]
	fmt.Println("ok:", ok)
	if !ok {
		//If no session cookie found or error occured, change username to zero string
		SentTemplate.Username = ""
		return
	}
	SentTemplate.Username = fmt.Sprint(username)
}

func BookHandler(w http.ResponseWriter, r *http.Request) {
	//Handler for book search function in main page
	r.ParseForm()
	searchName := r.FormValue("name")
	fmt.Println(searchName)
	//Find books from taken information that contains searched name
	//P.S. SentTemplate.Books that contains books info from database, it is taken in main function at the start of the server
	var Books []Book
	for _, bk := range SentTemplate.Books {
		if strings.Contains(strings.ToLower(bk.Name), strings.ToLower(searchName)) {
			book := bk
			Books = append(Books, book)
		}
	}
	Cookie(r) //Checking if logged in
	//New template, since we must not change info that taken from data
	SentTe := SentTemp{SentTemplate.Username, Books, searchName}
	tpl.ExecuteTemplate(w, "book.html", SentTe)
	if len(Books) == 0 {
		fmt.Println("No such book found")
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	//Delete session cookie and redirect to main page
	session, _ := store.Get(r, "session")
	session.Options.MaxAge = -1
	session.Save(r, w)
	http.Redirect(w, r, "/", http.StatusFound)
}

func MainPageHandler(w http.ResponseWriter, r *http.Request) {
	//Checking if logged in and running mainpage.html
	fmt.Println("Main page running")
	Cookie(r)
	tpl.ExecuteTemplate(w, "mainpage.html", SentTemplate)
}

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	//Running login.html with login form
	fmt.Println("Login handler successfully running")
	tpl.ExecuteTemplate(w, "login.html", nil)
}

func LoginConfirmationHadler(w http.ResponseWriter, r *http.Request) {
	//Function that checks if username and password are correct
	fmt.Println("Login confirmation successfully running")
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")
	//Taking hashed password from database with given username
	fmt.Println("Take success")
	fmt.Println(username, " ", password)
	var hash string
	statement := "SELECT password from Users WHERE username = ?"
	row := db.QueryRow(statement, username)
	fmt.Println("query success")
	err := row.Scan(&hash)
	if err != nil {
		fmt.Println("Error taking hash from db")
		tpl.ExecuteTemplate(w, "login.html", "Check username and password")
		return
	}
	//Comparing hashed password with written by user
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err == nil {
		//Username and password match with database, so create session cookie with given username
		fmt.Println("Login successfully")
		session, _ := store.Get(r, "session")
		session.Values["username"] = username
		session.Save(r, w)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	//If error occured, run login.html to try login again
	fmt.Println("Incorrect password")
	tpl.ExecuteTemplate(w, "login.html", "check username and password")
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	//Running register.html with registration form
	fmt.Println("Register handler successfully running")
	tpl.ExecuteTemplate(w, "register.html", nil)
}

func RegisterConfirmationHandler(w http.ResponseWriter, r *http.Request) {
	//Funtion that checks if username and password are valid, and inserts them in database
	fmt.Println("Register confirmation handler successfully running")
	r.ParseForm()
	username := r.FormValue("username")
	password := r.FormValue("password")
	//Validation of username
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
	//Validation of password
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
	//When both are valid, check if username already taken
	statement := "SELECT UserID from Users WHERE username = ?"
	row := db.QueryRow(statement, username)
	var UserID int
	err := row.Scan(&UserID)
	fmt.Println(UserID)
	if err != sql.ErrNoRows {
		tpl.ExecuteTemplate(w, "register.html", "Username already taken")
		return
	}
	//When username is unique, hash password and insert both in database
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
	_, err = insert.Exec(username, hash)
	if err != nil {
		tpl.ExecuteTemplate(w, "register.html", "Error inserting data")
		return
	}
	session, _ := store.Get(r, "session")
	session.Values["username"] = username
	session.Save(r, w)
	fmt.Println("User created successfully")
	http.Redirect(w, r, "/", http.StatusFound)
}
