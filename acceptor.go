package paxos

import (
   "encoding/json"
   "log"
   "net/http"
)

type Acceptor struct {
   PrevPropNum uint
   Store string
   Down bool
}

func NewAcceptor() Acceptor {
   return Acceptor {
      PrevPropNum: 0,
      Store: "",
      Down: false,
   }
}

func (me *Acceptor) BeAcceptor() {
   http.HandleFunc("/propose", me.Promise)
}

func (me *Acceptor) Promise(w http.ResponseWriter, r *http.Request) {
   var proposal map[string]interface{}
   err := json.NewDecoder(r.Body).Decode(&proposal)
   CheckError(err)

   var status int
   prevPropNum := me.PrevPropNum
   propNum := uint(proposal["propNum"].(float64))
   if propNum > me.PrevPropNum {
      me.PrevPropNum = propNum
      status = accept
   } else {
      status = reject
   }

   if !me.Down {
      promise := map[string]interface{} {
         "status": status,
         "prevPropNum": prevPropNum,
      }
      json.NewEncoder(w).Encode(&promise)

      switch status {
      case accept:
         log.Print("Accepted proposal ", propNum)
      case reject:
         log.Print("Rejected proposal ", propNum, " for ", me.PrevPropNum)
      default:
      }
   }
}

func (me *Acceptor) Commit(w http.ResponseWriter, r *http.Request) {
   // ignore commit if this acceptor is down
   if me.Down {
      return
   }

   // decode the commit packet
   var commit map[string]interface{}
   err := json.NewDecoder(r.Body).Decode(&commit)
   CheckError(err)
   propNum := uint(commit["propNum"].(float64))
   val := commit["val"].(string)

   // determine whether to accept or reject the commit
   var status int
   if propNum >= me.PrevPropNum {
      status = accept
      me.Store = val
      log.Print("Accepted commit ", propNum)
   } else {
      status = reject
      log.Print("Rejected commit ", propNum)
   }

   // reply to commit
   ack := map[string]interface{} {
      "status": status,
   }
   json.NewEncoder(w).Encode(&ack)
}
