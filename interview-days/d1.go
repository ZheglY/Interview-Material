package interviewdays

import (
	"fmt"
)

// создаем переменную без значения
// однако, имеет значение по умолчанию - zero value 
// age == 0
var age int

var age1 int = 25

var age2= 25

// Чаще всего используют именно zero value
func NewInt() int {
	age := 20
	return age
}

// Реальный кейс применения zero value

type User struct {
	name string
	age int
}

var user User

func NewUser() {
	user.name = "Ярослав"
	user.age = 69
}

// несколько переменных

var x, y int // 0, 0

func TwoPointers() (int, int){
	l, r := 0, 10
	return l, r
}

// Shadowing - перекрытие переменных

func D1() {
	x := 10

	if true {
		x := 20
		fmt.Println(x)
	}

	fmt.Println(x)
}

