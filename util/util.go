package util

import (
   "fmt"
)

func CheckError(err error) {
   if err != nil {
      panic(err)
   }
}

func FormUrl(sockAddr string, path string) string {
   return fmt.Sprintf("http://%s%s", sockAddr, path)
}

func Max(x uint, y uint) uint {
   if x > y {
      return x
   }
   return y
}

func IsMajority(accepts []string, numAcceptors int) bool {
   numAccepts := 0
   for _, accept := range accepts {
      if accept == "" {
         numAccepts += 1
      }
   }
   return numAccepts > numAcceptors / 2
}

//func IsMajority(confirmations []bool, numAcceptors int) bool {
   //numCommits := 0
   //for _, commit := range confirmations {
      //if commit {
         //numCommits += 1
      //}
   //}
   //return numCommits > numAcceptors / 2
//}

func RemoveRejects(votes []string) []string {
   var accepts []string

   for _, vote := range votes {
      if vote != "" {
         accepts = append(accepts, vote)
      }
   }

   return accepts
}
