![Logo](../logo.png)
  
### Intro
[g-fu](https://github.com/codr7/g-fu) is a pragmatic [Lisp](https://xkcd.com/297/) developed and embedded in Go with support for block-structured programming, quasi-quotation and macros, lambdas, optimized tail-recursion, opt-/varargs, first-class environments, user-defined setters, restarts, preemptive green threads, and channels.

### Status
The goal of version 1 is to nail syntax and semantics as far as possible without adding a compiler. It's not very [fast](https://github.com/codr7/g-fu/tree/master/v1/bench) yet, but mostly feature complete and correct as far as the [test suite](https://github.com/codr7/g-fu/tree/master/v1/test) goes. Besides bug fixes, I don't expect the code to change much moving forward.

```
$ git clone https://github.com/codr7/g-fu.git
$ cd g-fu/v1
$ go build src/gfu.go
$ rlwrap ./gfu
g-fu v1.21.0

Press Return twice to evaluate.

  (load "lib/all.gf")
```
```
  (let (foo (let (bar 42)
              Env/this))
    (say foo/bar)
    (use foo bar)
    (say bar))

42
42
```

### Syntax
g-fu quasi-quotes using `'` and splices using `%`, `_` is used for missing values and `..` to splat sequences.

### Vectors
g-fu uses vectors (or slices in Go lingo), rather than linked lists as a basic data structure.

The empty vector is written `()`, which is not the same thing as `_`.

```
  (len ())

0

  (len _)

Error: Len not supported: Nil
```

One consequence of using vectors is that items are pushed/popped last rather than first.

```
  (let (v ())
    (push v 1 2 3)
    (pop v)
    v)

(1 2)
```

### Types
`type` may be used to get the type of any value.

```
  (type 42)

Int

  (type Int)

Meta

  (type Meta)

Meta
```

Calling `X/?' for any type `X` and an arbitrary number of values and/or types returns the direct parent.

```
  (Int/? 42)
  
Int

  (Int/? T)
_

  (Seq/? Vec)

Seq

  (Seq/? IntIter)

Iter
```

### Conditions
```(load "lib/cond.gf")```

Every value has a boolean representation that may be retrieved using `bool`.

```
  (bool 42)

T

  (bool "")

F
```

Values may be combined using `or`/`and`, unused values are not evaluated. Comparisons are performed using boolean representations while preserving original values, the last evaluated argument is returned.

```
  (or 0 1)

1

  (or 0 F)
F

  (and 1 2 3)

3

  (and 1 _ 3)

_
```

`if` may be used to branch on a condition.

```
  (if 42 'foo 'bar)

'foo
```

The else-branch is optional.

```

  (if "" 'foo)  
_
```

`switch` may be used to combine multiple branches. Each branch is prefixed by its condition, and the first branch with a truthy condition is evaluated.

```
  (switch
    (F 'foo)
    (T 'bar)
    (T 'baz))

'bar
```

### Bindings
All identifiers except constants like `_`/`T`/`F` live in the same namespace. New bindings may be created using `let`.

`let` comes in two flavors. When called with arguments, it creates bindings and evaluates its body in a fresh environment.

```
  (let (foo 'outer)
    (let (foo 'inner)
      (say foo))
    (say foo))

inner
outer
```

And when called without arguments, it binds the specified names in the current environment instead. In the following example, `bar` and `baz` are bound in the current environment which already contains `foo`.

```
  (let (foo 1)
    (let bar 2 baz (+ bar 1))
    (say foo bar baz))

123
```

Shadowing is not allowed within the same environment.

```
  (let (foo 1)
    (let foo 2))

Error: Dup binding: foo 1
```

`set` may be used to change the value of existing bindings.

```
  (let (foo 1)
    (let _
      (set foo 3))
    (say foo))

3
```

### Environments
Environments are first class, `Env/this` evaluates to the current one.

Referenced bindings, such as the type `Env` from `Env/this` in the following example, are automatically captured.

```
  (let (foo 42) Env/this)

(foo:42 Env:Env)
```

Qualified identifiers allow reaching into external environments.

```
  (let (foo (let (bar 42) Env/this))
    foo/bar)

42
```

Since binding environments is a very common thing to do, `env` is provided as a shortcut.

```
(env foo (bar 42)
  (fun baz() (set bar 7)))
```
```
  foo/bar

42

  (foo/baz)
  foo/bar

7
```

Non-captured bindings may be imported manually.

```
(let _
  (env foo (bar 42))
  (env baz _ (use foo bar))
  baz/bar)

42
```

#### Setters
The `set`-protocol may be hooked into by binding `set-x` for any identifier `x`. The setter is called with an update-function and any number of keys and typically updates the value for the specified symbol at the specified key with the result of applying the update-function to the previous value.

```
  (let _
    (fun set-foo(f k..)
      (say k)
      (f 35))
      
    (inc (foo 'bar 'baz) 7))

bar baz
42
```

#### Proxies
Failed lookups may be trapped by defining `resolve`. The following example implements a basic proxy that forwards all lookups to the specified delegate. Since we're parameterizing the proxy, a function environment is used.

```
(fun proxy(d)
  (fun resolve(key)
    (d/val key))

  Env/this)
```
```
  (let (p (proxy (let (foo 42)
                   (use _ val)
                   Env/this)))
    p/foo)

42
```

#### Sandboxes
Environments may be used to isolate the evaluation of untrusted code.

The following example creates a sandbox named `sec` and imports `eval` and the local function `pub`. `use` is likewise imported, but restricted within eval.

```
(fun pub() (say 'pub))
(fun priv() (say 'priv))

(env sec _
  (use _ eval pub))
```

`pub` may be accessed from within `eval`.

```
  (sec/eval '(pub))

pub
```

While `priv` may not.

```
  (sec/eval '(priv))

panic: Error: Unknown: sec/priv
```

Importing doesn't work either.

```
  (sec/eval '(use _ priv))

panic: Error: Unknown: sec/priv
```

### Functions
Functions are created using `fun`; which accepts an optional name to bind in the current environment, a list of arguments and a body; and returns the function.

```
  (let _
    (fun say-hello(x) (say "Hello " x))
    (dump say-hello)
    (say-hello "World"))

(fun say-hello(x) (say "Hello " x))
Hello World
```

Arguments may be defined as optional by specifying default values.

```
  (let _
    (fun say-hello((x "World")) (say "Hello " x))
    (say-hello))

Hello World
```

Arguments suffixed with `..` consume any number of remaining values.

```
  (let _
    (fun say-hello(xs..) (say "Hello " xs))
    (say-hello 'Moe 'Larry 'Curly))

Hello Moe Larry Curly
```

The following example implements a simple counter as a closure.

```
  (let (i 0
        c (fun((d 1)) (inc i d)))
    (say (c 1))
    (say (c 2)))

1
3
```

### Macros
From a distance, macros look much like functions. They are defined the same way, accept arguments and return results. The difference is that macros have to deal with two dimensions, expansion time and evaluation time. As a consequence, macro arguments are not automatically evaluated. What is eventually evaluated is the result of expanding the macro.

The following example defines a macro called `foo` that expands to its argument.

```
  (let _
    (mac foo(x) x)
    (foo 42))

42
```

Raising the bar one notch, the `call`-macro below expands into code calling the specified target with arguments. `expand` may be used to get the resulting code from a macro call.

```
  (let _
    (mac call(x args..) '(%x %args..))
    (dump (expand 1 '(call + 35 7)))
    (call + 35 7))

(+ 35 7)
42
```

The next example is taken from the [standard library](https://github.com/codr7/g-fu/blob/master/v1/lib/abc.gf) and expands recursively to a nested series of calls using the previous result as the first argument.

```
(mac @(f1 fs..)
  '(fun(args..)
     %(tr fs '(call %f1 args..) (fun(acc x) '(call %x %acc)))))
```

#### Capture
When creating bindings in macro expansions, there is always a risk of capturing existing bindings.

```
  (let (foo 1)
    (mac bar(body..)
      '(let (foo 2) %body..))
    (bar (say foo)))

2
```

Prefixing any identifier with `$` binds the name to a unique symbol in the current environment unless already bound.

```
  $foo
  
foo-sym-168

  $foo
  
foo-sym-168
```

Unique symbols may be spliced into expansions to avoid capture.

```
  (let (foo 1)
    (mac bar(body..)
      '(let (%$foo 2) %body..))
    (bar (say foo)))

1
```

### Iterators
```(load "lib/iter.gf")```

Loops support exiting with a result using `(break ...)` and skipping to the start of next iteration using `(continue)`.

The fundamental loop is called `loop`, and that's exactly what it does until interrupted by `break` or the stack unwinds for some reason.

```
  (say (loop (say 'foo) (break 'bar) (say 'baz)))

foo
bar
```

`while` keeps iterating until the specified condition turns false.

```
  (let (i 0)
    (while (< i 3)
      (say (inc i))))

1
2
3
```

`for` accepts any iterable and an optional variable name, and runs one iteration for each value.

```
  (for 3 (say 'foo))

foo
foo
foo
```

```
  (for ('(foo bar baz) v) (say v))

foo
bar
baz
```

### Restarts
Restarts allow handling errors upstream without unwinding the call stack or requiring manual propagation.

```
  (fail "Going down")
  
Break: Error: Going down
0 abort
1 retry

Choose 0-1: 1

Break: Error: Going down
0 abort
1 retry

Choose 0-1: 0

Abort
```

`try` may be used to limit the scope and/or add custom restarts. The final value from the restart is returned from the try block except in case of `(retry)`/`(abort)`.

```
  (try ((foo (x)
          (say (str "foo " x))
          'bar))
    (fail "Going down"))

Break: Error: Going down
0 abort
1 retry
2 foo x

Choose 0-2: 2 42

foo 42
bar
```

`restart` may be used to look up restarts in the current call stack.

```
  (try ((foo (x) (+ x 35)))
    (try _
      (call (restart 'foo) 7)))

42
```

Restarts live in a separate namespace, but allow shadowing just like regular bindings.

```
  (try ((foo () 'bar))
    (try ((foo () 'baz))
      (call (restart 'foo))))
  
baz
```

`(abort)` exits unconditionally without entering a break loop.

```
  (try _ (abort) (say "Not bloody likely"))

Abort
```

A more realistic scenario may be triggered by loading a nonexistent file, which includes a restart for using a different filename.

test.gf
```
42
```

```
(load "not.found")

Break: Error: Failed loading file: "not.found"
open not.found: no such file or directory
0 abort
1 retry
2 use-filename new

Choose 0-2: 2 "test.gf"

42
```

If you still can't see the point, imagine an expensive computation before the `load` that is used after.

```
  (do
    (say "Expensive computation")
    (load "not.found")
    (say "Use result of computation"))      
  
Expensive computation

Break: Error: Failed loading file: "not.found"
open not.found: no such file or directory
0 abort
1 retry
2 use-filename new

Choose 0-2: 2 "test.gf"

Use result of computation
```

### Tasks
Tasks are first class, preemptive green threads (or goroutines) that run in separate environments and interact with the outside world using channels. New tasks are started using `task`, which takes an optional task id and channel or buffer size and returns the task. `wait` may be used to sleep until a set of tasks are done and get the results.

```
  (let _
    (task t1 () (say 'foo) 'bar)
    (task t2 () (say 'baz) 'qux)
    (say (wait t1 t2)))

baz
foo
bar qux
```

The defining environment is cloned.

```
  (let (v 42)
    (say (wait (task () (inc v))))
    (say v))

43
42
```

#### Channels
Channels are optionally buffered thread-safe pipes. `chan` may be used to create new channels, and `push`/`pop` to transfer values. `len` returns the current number of buffered values.

```
  (let (c (Chan/new 1))
    (push c 42)
    (say (len c))
    (say (pop c)))

1
42
```

Unbuffered channels are useful for synchronizing tasks. The following example starts a task called `t` and puts the current task in its inbox, `t` then replies `'foo` and returns `'bar`.

```
  (let _
    (task t ()
      (Task/post (fetch) 'foo)
      'bar)
      
    (t/post Task/this)
    (say (fetch))
    (say (wait t)))

foo
bar
```

### Profiles
CPU profiling may be enabled by passing `-prof` on the command line; results are written to the specified file, `fib_tail.prof` in the following example.

```
$ gfu -prof fib_tail.prof bench/fib_tail.gf

$ go tool pprof fib_tail.prof
File: gfu
Type: cpu
Time: Apr 12, 2019 at 12:52am (CEST)
Duration: 16.52s, Total samples = 17.31s (104.79%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top10
Showing nodes accounting for 10890ms, 62.91% of 17310ms total
Dropped 99 nodes (cum <= 86.55ms)
Showing top 10 nodes out of 79
      flat  flat%   sum%        cum   cum%
    2130ms 12.31% 12.31%    15980ms 92.32%  _/home/a/Dev/g-fu/v1/src/gfu.Vec.EvalVec
    1580ms  9.13% 21.43%    15980ms 92.32%  _/home/a/Dev/g-fu/v1/src/gfu.(*VecType).Eval
    1500ms  8.67% 30.10%     1680ms  9.71%  runtime.heapBitsSetType
    1180ms  6.82% 36.92%     1520ms  8.78%  _/home/a/Dev/g-fu/v1/src/gfu.(*Env).Find
     970ms  5.60% 42.52%     2290ms 13.23%  _/home/a/Dev/g-fu/v1/src/gfu.(*SymType).Eval
     830ms  4.79% 47.31%    15980ms 92.32%  _/home/a/Dev/g-fu/v1/src/gfu.Val.Eval
     780ms  4.51% 51.82%     4440ms 25.65%  runtime.mallocgc
     770ms  4.45% 56.27%      770ms  4.45%  runtime.memclrNoHeapPointers
     730ms  4.22% 60.49%    15980ms 92.32%  _/home/a/Dev/g-fu/v1/src/gfu.(*FunType).Call
     420ms  2.43% 62.91%      420ms  2.43%  runtime.memmove
```

### License
[MIT](https://github.com/codr7/g-fu/blob/master/LICENSE.txt)

### support
Please consider a donation if you would like to support the project, every contribution helps.

<a href="https://liberapay.com/codr7/donate"><img alt="Donate using Liberapay" src="https://liberapay.com/assets/widgets/donate.svg"></a>