package paxos

import (
   "bytes"
   "encoding/json"
   "log"
   "net/http"
   "sync"
)

type Proposer struct {
   Id uint
   Acceptors []string
   PropNum uint
   PrevPropNum uint
}

type proposal struct {
   propNum uint
   val uint
}

func NewProposer(id uint, acceptors []string) Proposer {
   return Proposer {
      Id: id,
      Acceptors: acceptors,
      PrevPropNum: 0,
   }
}

func (me *Proposer) BeProposer() {
   http.HandleFunc("/set", me.Propose)
   http.HandleFunc("/read", me.Read)
}

func (me *Proposer) Propose(w http.ResponseWriter, r *http.Request) {
   var val string
   err := json.NewDecoder(r.Body).Decode(&val)
   CheckError(err)

   log.Print("Proposing value ", val)
   acceptors := me.Prepare()
   me.Accept(acceptors, val)
}

func (me *Proposer) Read(w http.ResponseWriter, r *http.Request) {
   log.Print("reading")
   numAcceptors := len(me.Acceptors)
   var wg sync.WaitGroup
   wg.Add(numAcceptors)

   // ask each acceptor for their value
   vals := make([]string, numAcceptors)
   for i, acceptor := range me.Acceptors {
      go func(i int, acceptor string) {
         defer wg.Done()

         // get the value acceptor is storing
         url := FormUrl(acceptor, "/retrieve")
         resp, err := http.Get(url)
         defer resp.Body.Close()

         // acceptor is down
         if err != nil {
            log.Print(err)
            vals[i] = ""
         }

         // decode acceptor's response
         var val string
         json.NewDecoder(resp.Body).Decode(&val)
         log.Print(val)

         vals[i] = val
      }(i, acceptor)
   }

   // wait for all acceptors to respond
   wg.Wait()
   log.Print(vals)

   // get frequencies of the values the acceptors gave us
   freqs := make(map[string]int)
   for _, val := range vals {
      if _, ok := freqs[val]; ok {
         freqs[val] += 1
      } else {
         freqs[val] = 1
      }
   }

   // find the max frequency and the most common value
   maxFreq := 0
   mostFreqVal := ""
   for val, freq := range freqs {
      if freq > maxFreq {
         maxFreq = freq
         mostFreqVal = val
      }
   }

   // determine whether a majority of acceptors
   // have replied with the same value
   var status int
   if maxFreq > numAcceptors / 2 {
      status = accept
   } else {
      status = reject
   }

   // reply to request
   respData := map[string]interface{} {
      "status": status,
      "val": mostFreqVal,
   }
   json.NewEncoder(w).Encode(respData)
}

/*
 *sends proposals until a majority has been reached and
 *returns a list of socket addresses of acceptors who accepted
 */
func (me *Proposer) Prepare() []string {
   var accepts []string
   accepted := false

   // send proposals until a majority has been reached
   for !accepted {
      me.PropNum = me.getPropNum(me.PrevPropNum)

      // send the proposal to all acceptors
      votes := me.sendProposal()
      accepts = RemoveRejects(votes)

      // calculate whether a majority has been reached
      accepted = (len(accepts) + 1) > (len(me.Acceptors) + 1) / 2

      if accepted {
         log.Print("Proposal ", me.PropNum,
               " accepted by a majority of acceptors")
      } else {
         log.Print("Proposal ", me.PropNum, " rejected... retrying")
      }
   }

   return accepts
}

 /*
  *sends proposal to all acceptors and
  *returns list with socket addresses for accepts
  *and empty strings for rejects
  */
func (me *Proposer) sendProposal() []string {
   // list with socket addresses for accepts
   // and empty strings for rejects
   accepts := make([]string, len(me.Acceptors))

   var wg sync.WaitGroup
   wg.Add(len(me.Acceptors))

   // send proposal to all acceptors
   log.Print(me.Acceptors)
   for i, acceptor := range me.Acceptors {
      go func(i int, acceptor string) {
         defer wg.Done()

         // create the proposal
         proposal := map[string]interface{} {
            "propNum": me.PropNum,
            "val": me.Id,
         }

         // get the proposal ready to send
         encoded, err := json.Marshal(proposal)
         CheckError(err)
         payload := bytes.NewBuffer(encoded)

         // send proposal to acceptors[i]
         url := FormUrl(acceptor, "/propose")
         log.Print("Sent proposal to ", url)
         resp, err := http.Post(url, "application/json", payload)

         // acceptor is down
         if err != nil {
            panic(err)
            return
         }

         // decode promise from acceptor
         var promise Promise
         json.NewDecoder(resp.Body).Decode(&promise)

         // interpret promise
         switch promise.status {
         case accept:
            accepts[i] = acceptor
         case reject:
         default:
            panic("Unsupported promise status")
         }

         me.PrevPropNum = Max(me.PrevPropNum, promise.prevPropNum)
      }(i, acceptor)
   }

   // wait for all promises to be received
   wg.Wait()

   return accepts
}

func (me *Proposer) Accept(acceptors []string, val string) bool {
   commits := make([]bool, len(acceptors))

   var wg sync.WaitGroup
   wg.Add(len(acceptors))

   for i, acceptor := range acceptors {
      go func(i int, acceptor string) {
         defer wg.Done()

         // get the commit message ready to send
         commit := map[string]interface{} {
            "propNum": me.PropNum,
            "val": val,
         }
         encoded, err := json.Marshal(commit)
         CheckError(err)
         payload := bytes.NewBuffer(encoded)

         // send commit message to acceptors[i]
         url := FormUrl(acceptor, "/commit")
         resp, err := http.Post(url, "application/json", payload)

         if err != nil {
            commits[i] = false
            return
         }

         var confirmation Committed
         json.NewDecoder(resp.Body).Decode(&confirmation)

         switch confirmation.status {
         case accept:
            commits[i] = true
         case reject:
            commits[i] = false
         default:
            panic("Unsupported commit status")
         }
      }(i, acceptor)
   }

   wg.Wait()

   return true
}

// returns the next highest multiple of this proposer's id
func (me *Proposer) getPropNum(prevPropNum uint) uint {
   log.Print("my propNum: ", me.Id)
   propNum := (prevPropNum / me.Id) * me.Id

   var i uint = 0
   for i <= prevPropNum % me.Id {
      i += me.Id
   }
   propNum += i

   return propNum
}
