package main

import 	(
	"log"
	"golang.org/x/crypto/bcrypt"
)


// Remember this is intended for in person use or playing online with close friends
// It has little security, this check is intended to deter players from cheating
// and to keep the app a little bit more organized.
func CheckPassword(password string, hash []byte) bool {

	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if  err != nil {
		log.Println("Login attempt failed: ", err.Error())
		return false
	} 

	return true
}

func CreateHash(pass string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(pass), 10)
}