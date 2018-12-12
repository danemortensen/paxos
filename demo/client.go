package main

import (
   "bufio"
   "bytes"
   "encoding/json"
   "fmt"
   "net/http"
   "os"
   "strings"

   "github.com/danemortensen/paxos"
)

func instruct() {
   fmt.Println("Usage:")
   fmt.Println("\tSet value: set <proposer-sock-addr> <value>")
   fmt.Println("\tGet value: get <proposer-sock-addr>")
   fmt.Println("\tHelp: help")
   fmt.Println("\tQuit: quit")
}

func setValue(proposer string, val string) {
   encoded, err := json.Marshal(val)
   paxos.CheckError(err)
   payload := bytes.NewBuffer(encoded)

   url := paxos.FormUrl(proposer, "/set")
   resp, err := http.Post(url, "application/json", payload)
   defer resp.Body.Close()
   if err != nil {
      fmt.Println("node not available")
   }
}

func checkArgs(args []string, argc int) bool {
   return len(args) == argc
}

func interpret(input string) {
   words := strings.Split(input, " ")

   switch words[0] {
   case "set":
      if checkArgs(words, 3) {
         setValue(words[1], words[2])
      }
   case "get":
   case "help":
      instruct()
   default:
      fmt.Println("Invalid input")
   }
}

func main() {
   instruct()
   scanner := bufio.NewScanner(os.Stdin)
   for {
      fmt.Printf("Enter a command: ")
      scanner.Scan()
      input := scanner.Text()
      if input == "quit" {
         break
      }
      interpret(input)
   }
   paxos.CheckError(scanner.Err())
}
