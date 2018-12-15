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
   fmt.Println("\tWrite value: write <proposer-sock-addr> <value>")
   fmt.Println("\tGet value: get <proposer-sock-addr>")
   fmt.Println("\tHelp: help")
   fmt.Println("\tQuit: quit")
}

func write(proposer string, val string) {
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

func read(proposer string) {
   url := paxos.FormUrl(proposer, "/read")
   resp, err := http.Get(url)
   if err != nil {
      fmt.Println("node not available")
      return
   }
   defer resp.Body.Close()

   var respData map[string]interface{}
   json.NewDecoder(resp.Body).Decode(&respData)
   switch int(respData["status"].(float64)) {
   case 0:
      fmt.Println("Stored value: ", respData["val"].(string))
   case 1:
      fmt.Println("Error: consensus not reached")
   default:
      fmt.Println("lookitup")
   }
}

func checkArgs(args []string, argc int) bool {
   return len(args) == argc
}

func interpret(input string) {
   words := strings.Split(input, " ")

   switch words[0] {
   case "write":
      if checkArgs(words, 3) {
         write(words[1], words[2])
      }
   case "get":
      if checkArgs(words, 2) {
         read(words[1])
      }
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
