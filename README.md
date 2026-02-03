goexif
======

[![GoDoc](https://godoc.org/github.com/rwcarlsen/goexif?status.svg)](https://godoc.org/github.com/rwcarlsen/goexif)

Fork of the original [goexif](https://github.com/rwcarlsen/goexif)... just less inefficient (4-5x speedup/less allocations).

```
          │ initial.txt  │                new.txt                 │
          │    sec/op    │   sec/op     vs base                   │
Decode-16   3844.6µ ± 3%   944.1µ ± 2%  -75.44% (p=0.000 n=10+20)

          │  initial.txt  │             new.txt             │
          │     B/op      │     B/op      vs base           │
Decode-16   13.541Mi ± 0%   1.805Mi ± 0%  -86.67% (n=10+20)

          │ initial.txt  │            new.txt             │
          │  allocs/op   │  allocs/op   vs base           │
Decode-16   29.675k ± 0%   7.011k ± 0%  -76.37% (n=10+20)
```
