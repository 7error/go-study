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
	"fmt"
	"io/ioutil"
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

	// create an empty != nil user to store the final user
	out := &user{}

	content, _ := ioutil.ReadFile("user.yaml")
	content = []byte(os.ExpandEnv(string(content)))

	_ = yaml.Unmarshal(content, out)

	fmt.Printf("\n%v\n", out)
}
