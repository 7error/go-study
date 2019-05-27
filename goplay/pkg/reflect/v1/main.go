// Copyright 2019 Yandy Ramirez
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"fmt"
	"reflect"
)

// user is the type we're playing with
type user struct {
	ID    string
	Name  string
	Email string
}

// String implements the Stringer using the reflect package
func (u *user) String() string {
	v := reflect.ValueOf(*u)

	buf := &bytes.Buffer{}
	for i := 0; i < v.NumField(); i++ {
		if i > 0 {
			buf.WriteByte(' ')
		}
		fmt.Fprintf(buf, "(%s: %v)", v.Type().Field(i).Name, v.Field(i))
	}

	return buf.String()
}

func main() {

	in := &user{
		ID:    "1234",
		Name:  "ME TWO",
		Email: "LOL@LMAFO.ROFL",
	}

	t := reflect.TypeOf(in)

	fmt.Println()       //
	fmt.Printf("%v", t) // Output: *main.user
	fmt.Println()       //

	v := reflect.ValueOf(in) //
	fmt.Println()            //
	fmt.Printf("%v", v)      //
	// Output:
	//	id: "1234"
	// 	name: ME TWO
	//	email: LOL@LMAFO.ROFL

	fmt.Println()           //
	fmt.Println(v.String()) // Output: <*main.user Value>

}
