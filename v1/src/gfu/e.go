package gfu

import (
  "bufio"
  "fmt"
  //"log"
)

type E interface {
  Dumper
}

type BasicE struct {
  msg string
}

func (e *BasicE) Init(g *G, msg string) *BasicE {
  if g.Debug {
    panic(msg)
  }

  e.msg = msg
  return e
}

func (e *BasicE) Dump(g *G, out *bufio.Writer) E {
  out.WriteString(e.msg)
  return nil
}

func (g *G) E(msg string, args ...interface{}) *BasicE {
  for i, a := range args {
    switch v := a.(type) {
    case Dumper:
      var e E

      if args[i], e = g.DumpString(v); e != nil {
        args[i] = e
      }
    case Val:
      var e E

      if args[i], e = g.String(v); e != nil {
        args[i] = e
      }
    }
  }

  msg = fmt.Sprintf(msg, args...)
  e := new(BasicE).Init(g, fmt.Sprintf("Error: %v", msg))

  if g.Debug {
    s, _ := g.DumpString(e)
    panic(s)
  }

  return e
}

type ReadE struct {
  BasicE
  pos Pos
}

func (e *ReadE) Init(g *G, pos Pos, msg string) *ReadE {
  e.BasicE.Init(g, msg)
  e.pos = pos
  return e
}

func (e *ReadE) Dump(g *G, out *bufio.Writer) E {
  p := &e.pos

  fmt.Fprintf(out,
    "Read error in '%s'; row %v, col %v:\n%v",
    p.src, p.Row, p.Col, e.msg)

  return nil
}

func (g *G) ReadE(pos Pos, msg string, args ...interface{}) *ReadE {
  msg = fmt.Sprintf(msg, args...)
  e := new(ReadE).Init(g, pos, msg)
  return e
}
