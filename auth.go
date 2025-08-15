package main

import 	(
	"strings"
	b "encoding/base64"
	"fmt"
)


// Remember this is intended for in person use or playing online with close friends
// It has no security, this check is intended to deter players from cheating
// and to keep the app a little bit more organized.
func CheckPassword(h string, i uint64, p string) bool {
	n:=len(strings.Trim(h," "))*int(i)
	pp:=b.URLEncoding.EncodeToString([]byte(fmt.Sprintf("%d %s %d", n, strings.Trim(h," "), i)))
	if pp != p {
		fmt.Println("Password is supposed to be ", pp)
	}
	return pp == p
}