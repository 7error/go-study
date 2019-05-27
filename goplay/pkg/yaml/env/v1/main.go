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
	"html/template"
	"os"

	"github.com/joho/godotenv"
	yaml "gopkg.in/yaml.v2"
)

// init loads a local .env file if present
func init() {
	_ = godotenv.Load()
}

// user os the type we're playing with
type user struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// String implements the Stringer interface to use with fmt
func (u *user) String() string {
	b, _ := yaml.Marshal(u)

	return string(b)
}

func main() {

	// First we read in the ENV variables with the os package
	in := &user{
		ID:    os.Getenv("ID"),
		Name:  os.Getenv("NAME"),
		Email: os.Getenv("EMAIL"),
	}

	// create an empty != nil user to store the final user
	out := &user{}

	// parse the template file
	tpl, _ := template.ParseFiles("user.tpl.yaml")

	// create a empty != nil buffer to store the Executed template
	buf := &bytes.Buffer{}

	// execute the template and store the output in the buf variable
	_ = tpl.Execute(buf, in)

	// unmarshal the bytes from the buffer back into the out variable
	// there has to be an easier way.
	_ = yaml.Unmarshal(buf.Bytes(), out)

	fmt.Printf("\n%v\n", out)
}
