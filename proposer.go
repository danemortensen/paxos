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
}

func (me *Proposer) Propose(w http.ResponseWriter, r *http.Request) {
   var val string
   err := json.NewDecoder(r.Body).Decode(&val)
   CheckError(err)

   log.Print("Proposing value ", val)
   accepts := me.Prepare()
   me.Accept(acceptors, val)
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

   //return IsMajority(commits, len(me.numAcceptors))
   return true
}

// returns the next highest multiple of this proposer's id
func (me *Proposer) getPropNum(prevPropNum uint) uint {
   propNum := (prevPropNum / me.Id) * me.Id

   var i uint = 0
   for i <= prevPropNum % me.Id {
      i += me.Id
   }
   propNum += i

   return propNum
}
