package main

import (
   "bytes"
   "encoding/json"
   "flag"
   "fmt"
   "log"
   "math/big"
   "net"
   "net/http"

   "github.com/danemortensen/paxos"
)

type Node struct {
   Id uint
   Host string
   Port int
   Peers []string
   Failing bool
   PropNum uint
   HighestId uint

   paxos.Proposer
   paxos.Acceptor
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
   id uint
   peers []string
}

func (me *Node) connect(contact string) {
   // send my socket address to the provided contact
   url := formUrl(contact, "/join")
   encoded, err := json.Marshal(me.getSockAddr())
   checkError(err)
   payload := bytes.NewBuffer(encoded)
   resp, err := http.Post(url, "application/json", payload)
   checkError(err)
   defer resp.Body.Close()

   var respData map[string]interface{}
   json.NewDecoder(resp.Body).Decode(&respData)
   me.Id = uint(respData["id"].(float64))
   me.Proposer.Id = me.Id
   for _, peer := range respData["peers"].([]interface{}) {
      me.Peers = append(me.Peers, peer.(string))
      log.Print("Added peer ", peer)
   }
   me.Proposer.Acceptors = me.Peers
}

// add a new node and update peers
func (me *Node) addNode(w http.ResponseWriter, r *http.Request) {
   // obtain new node's socket address from request
   var newNode string
   err := json.NewDecoder(r.Body).Decode(&newNode)
   checkError(err)

   // respond with my socket address appended to peer list
   me.HighestId = nextPrime(me.HighestId)
   respData := map[string]interface{} {
      "id": me.HighestId,
      "peers": append(me.Peers, me.getSockAddr()),
   }
   json.NewEncoder(w).Encode(respData)

   for _, peer := range me.Peers {
      go func(peer string) {
         // send new node's socket address to peers
         nodeData := map[string]interface{} {
            "id": me.HighestId,
            "sockAddr": newNode,
         }
         encoded, err := json.Marshal(nodeData)
         checkError(err)
         payload := bytes.NewBuffer(encoded)

         url := formUrl(peer, "/add")
         resp, err := http.Post(url, "application/json", payload)
         checkError(err)
         resp.Body.Close()
      }(peer)
   }

   // add new node's socket address to my peer list
   me.Peers = append(me.Peers, newNode)
   me.Proposer.Acceptors = me.Peers
   log.Print("Added peer ", newNode)
}

// update peer list when told about a new node
func (me *Node) updatePeerList(w http.ResponseWriter, r *http.Request) {
   // obtain new node's socket address from request
   var nodeData map[string]interface{}
   err := json.NewDecoder(r.Body).Decode(&nodeData)
   checkError(err)

   // add new node's socket address to my peer list
   me.HighestId = uint(nodeData["id"].(float64))
   me.Peers = append(me.Peers, nodeData["sockAddr"].(string))
   me.Proposer.Acceptors = me.Peers
   log.Print("Added peer ", nodeData["sockAddr"])
}

func main() {
   flag.Parse()
   if contact == "" {
      me.Id = 1
      me.PropNum = 1
      me.HighestId = 1
   } else {
      go me.connect(contact)
   }

   http.HandleFunc("/join", me.addNode)
   http.HandleFunc("/add", me.updatePeerList)

   me.Proposer = paxos.NewProposer(me.Id, me.Peers)
   me.Acceptor = paxos.NewAcceptor()
   me.BeProposer()
   me.BeAcceptor()

   listener, err := net.Listen("tcp", me.getSockAddr())
   checkError(err)
   log.Print("Node running on ", me.getSockAddr())
   log.Fatal(http.Serve(listener, nil))
}
