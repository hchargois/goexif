goexif
======

[![GoDoc](https://godoc.org/github.com/hchargois/goexif?status.svg)](https://godoc.org/github.com/hchargois/goexif)

Fork of the original [goexif](https://github.com/rwcarlsen/goexif)... just less inefficient (5x faster, almost 10x less allocations).

```
                           │ initial.txt  │               new.txt               │
                           │    sec/op    │   sec/op     vs base                │
Decode/ReadNoTags-16         4019.3µ ± 2%   779.3µ ± 1%  -80.61% (p=0.000 n=20)
Decode/ReadAllToStrings-16    6.492m ± 1%   1.165m ± 3%  -82.06% (p=0.000 n=20)
Decode/ReadAllToTypes-16     4043.6µ ± 2%   848.6µ ± 1%  -79.01% (p=0.000 n=20)
geomean                       4.725m        916.8µ       -80.60%

                           │  initial.txt  │               new.txt                │
                           │     B/op      │     B/op      vs base                │
Decode/ReadNoTags-16         13.541Mi ± 0%   1.502Mi ± 0%  -88.91% (p=0.000 n=20)
Decode/ReadAllToStrings-16   14.157Mi ± 0%   1.816Mi ± 0%  -87.18% (p=0.000 n=20)
Decode/ReadAllToTypes-16     13.541Mi ± 0%   1.508Mi ± 0%  -88.86% (p=0.000 n=20)
geomean                       13.74Mi        1.602Mi       -88.34%

                           │ initial.txt  │               new.txt               │
                           │  allocs/op   │  allocs/op   vs base                │
Decode/ReadNoTags-16         29.675k ± 0%   3.990k ± 0%  -86.55% (p=0.000 n=20)
Decode/ReadAllToStrings-16   41.938k ± 0%   6.266k ± 0%  -85.06% (p=0.000 n=20)
Decode/ReadAllToTypes-16     29.675k ± 0%   4.355k ± 0%  -85.32% (p=0.000 n=20)
geomean                       33.30k        4.775k       -85.66%
```
