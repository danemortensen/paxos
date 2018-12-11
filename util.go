package paxos/util

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
