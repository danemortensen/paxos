package paxos

const (
   accept = iota
   reject = iota
)

type Promise struct {
   status int
   prevPropNum uint
}

type Commit struct {
   PropNum uint
   Val string
}

type Committed struct {
   status int
   propNum uint
}
