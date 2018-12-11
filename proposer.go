package paxos

import (
   "bytes"
   "encoding/json"
   "net/http"
   "sync"

   "github.com/danemortensen/paxos/util"
)

type proposer struct {
   id uint
   acceptors []string
   propNum uint
   prevPropNum uint
}

type proposal struct {
   propNum uint
   val uint
}

func NewProposer(id uint, acceptors []string) proposer {
   return proposer {
      id: id,
      acceptors: acceptors,
      prevPropNum: 0,
   }
}

func (me *proposer) Propose() {
   //accepts := me.Prepare()
}


/*
 *sends proposals until a majority has been reached and
 *returns a list of socket addresses of acceptors who accepted
 */
func (me *proposer) Prepare() []string {
   var accepts []string
   accepted := false

   // send proposals until a majority has been reached
   for !accepted {
      me.propNum = me.getPropNum(me.prevPropNum)

      // create the proposal
      proposal := proposal {
         propNum: me.propNum,
         val: me.id,
      }

      // send the proposal to all acceptors
      votes := me.sendProposal(&proposal)
      accepts = util.RemoveRejects(votes)

      // calculate whether a majority has been reached
      accepted = len(accepts) > len(me.acceptors) / 2
   }


   return accepts
}

 /*
  *sends proposal to all acceptors and
  *returns list with socket addresses for accepts
  *and empty strings for rejects
  */
func (me *proposer) sendProposal(proposal *proposal) []string {
   // get the proposal ready to send
   encoded, err := json.Marshal(*proposal)
   util.CheckError(err)
   payload := bytes.NewBuffer(encoded)

   // list with socket addresses for accepts
   // and empty strings for rejects
   accepts := make([]string, len(me.acceptors))

   var wg sync.WaitGroup
   wg.Add(len(me.acceptors))

   // send proposal to all acceptors
   for i, acceptor := range me.acceptors {
      go func() {
         defer wg.Done()

         // send proposal to acceptors[i]
         url := util.FormUrl(acceptor, "/propose")
         resp, err := http.Post(url, "application/json", payload)

         // acceptor is down
         if err != nil {
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

         me.prevPropNum = util.Max(me.prevPropNum, promise.prevPropNum)
      }()
   }

   // wait for all promises to be received
   wg.Wait()

   return accepts
}

func (me *proposer) Accept(acceptors []string, val string) bool {
   // get the commit message ready to send
   commit := Commit {
      PropNum: me.propNum,
      Val: val,
   }
   encoded, err := json.Marshal(commit)
   util.CheckError(err)
   payload := bytes.NewBuffer(encoded)

   commits := make([]bool, len(acceptors))

   var wg sync.WaitGroup
   wg.Add(len(acceptors))

   for i, acceptor := range acceptors {
      go func() {
         defer wg.Done()

         // send commit message to acceptors[i]
         url := util.FormUrl(acceptor, "/commit")
         resp, err := http.Post(url, "application/json", )

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
         break:
            panic("Unsupported commit status")
         }
      }()
   }

   return util.IsMajority(commits, me.numAcceptors)
}

// returns the next highest multiple of this proposer's id
func (me *proposer) getPropNum(prevPropNum uint) uint {
   propNum := (prevPropNum / me.id) * me.id

   var i uint = 0
   for i <= prevPropNum % me.id {
      i += me.id
   }
   propNum += i

   return propNum
}
