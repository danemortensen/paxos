package main

import (
   "bufio"
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "math/big"
   "net"
   "net/http"
   "os"
)

type Node struct {
   Id uint
   Host string
   Port int
   Peers []string
   Failing bool
   PropNum uint
   HighestId uint
}

var (
   me Node
   contact string
)

func (n *Node) getSockAddr() string {
   return fmt.Sprintf("%s:%d", n.Host, n.Port)
}

func formUrl(sockAddr string, path string) string {
   return fmt.Sprintf("http://%s%s", sockAddr, path)
}

func nextPrime(lower uint) uint {
   var maybePrime uint = lower + (1 + lower % 2)

   for !big.NewInt(int64(maybePrime)).ProbablyPrime(20) {
      maybePrime += 2
   }

   return maybePrime
}

func init() {
   flag.StringVar(&me.Host, "host", "localhost", "url of node")
   flag.IntVar(&me.Port, "port", 3000, "port of node")
   flag.StringVar(&contact, "contact", "",
         "socket address of a node in the system")
}

func checkError(err error) {
   if err != nil {
      log.Fatal(err)
   }
}

type connResp struct {
   Id uint
   Peers []string
}

func (me *Node) connect(contact string) {
   // send my socket address to the provided contact
   url := formUrl(contact, "/join")
   encoded, err := json.Marshal(url)
   checkError(err)
   payload := bytes.NewBuffer(encoded)
   resp, err := http.Post(url, "application/json", payload)
   checkError(err)

   // add the peers from the response to my peer list
   var respData struct {
      id uint
      peers []string
   }
   json.NewDecoder(resp.Body).Decode(&respData)
   me.Id = respData.id
   for _, peer := range respData.peers {
      me.Peers = append(me.Peers, peer)
   }
   resp.Body.Close()
}

// add a new node and update peers
func (me *Node) addNode(w http.ResponseWriter, r *http.Request) {
   fmt.Println("adding node")
   // obtain new node's socket address from request
   var newNode string
   err := json.NewDecoder(r.Body).Decode(&newNode)
   checkError(err)

   // respond with my socket address appended to peer list
   me.HighestId = nextPrime(me.HighestId)
   respData := struct {
      id uint
      peers []string
   }{
      me.HighestId,
      append(me.Peers, me.getSockAddr()),
   }
   json.NewEncoder(w).Encode(respData)

   // send new node's socket address to peers
   nodeData := struct {
      id uint
      sockAddr string
   }{
      me.HighestId,
      newNode,
   }
   encoded, err := json.Marshal(nodeData)
   checkError(err)
   payload := bytes.NewBuffer(encoded)
   for _, peer := range me.Peers {
      go func() {
         url := peer + "/add"
         resp, err := http.Post(url, "application/json", payload)
         checkError(err)
         resp.Body.Close()
      }()
   }

   // add new node's socket address to my peer list
   me.Peers = append(me.Peers, newNode)
}

// update peer list when told about a new node
func (me *Node) updatePeers(w http.ResponseWriter, r *http.Request) {
   // obtain new node's socket address from request
   var nodeData struct {
      id uint
      sockAddr string
   }
   err := json.NewDecoder(r.Body).Decode(&nodeData)
   checkError(err)

   // add new node's socket address to my peer list
   me.HighestId = nodeData.id
   me.Peers = append(me.Peers, nodeData.sockAddr)
}

func (me *Node) getInput() {
   scanner := bufio.NewScanner(os.Stdin)
   var input string

   for {
      fmt.Printf("Enter a command: ")
      scanner.Scan()
      input = scanner.Text()
      fmt.Println(input)
   }
}

func main() {
   fmt.Println("starting")
   flag.Parse()
   if contact == "" {
      me.Id = 1
      me.PropNum = 1
      me.HighestId = 1
   } else {
      go me.connect(contact)
   }

   http.HandleFunc("/join", me.addNode)
   http.HandleFunc("/add", me.updatePeers)

   listener, err := net.Listen("tcp", me.getSockAddr())
   checkError(err)
   go me.getInput()
   log.Fatal(http.Serve(listener, nil))
}
