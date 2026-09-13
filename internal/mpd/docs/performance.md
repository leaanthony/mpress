# M-Press Flavoured Markdown parser performance

This report records the first native M-Press Flavoured Markdown parser implementation and
the optimization work completed on 11 August 2026.

## Test system

- Revision before the parser work: `859b183`
- Go: `go1.26.5 linux/amd64`
- CPU: AMD Ryzen 9 3950X, 16 cores and 32 logical CPUs
- Kernel: Linux 7.1.3 x86-64
- Benchmark corpus: 10,000 deterministic Markdown pages and 256 referenced SVG
  assets
- Markdown source: 8,694,682 bytes
- Asset and shared-include data: 118,156 bytes
- Corpus SHA-256: `98e9e0d411825058c03395b49ac7f18e0fb478e6379a6c193a37773f610a6ba3`

The corpus includes ten page shapes. Together they exercise metadata, prose,
inline syntax, code, lists, tasks, nested quotes, tables, reference links,
footnotes, imports, image assets, and nested components.

Generate and verify it with:

```sh
go run ./cmd/mpd-corpus -output /tmp/mpd-corpus -pages 10000 -assets 256
```

The command reads every generated file again, recomputes the manifest digest,
parses all 10,000 pages, rejects parser errors, and confirms every generated
asset.

## Initial baseline

The initial parser already used an arena tree and immutable source slices, but
calculated every node position by scanning from byte zero. Ten one-iteration
samples took approximately 628 to 726 ms per corpus, with a median of 693 ms.
It allocated approximately 93,194,000 bytes and 260,000 objects per corpus.

The first CPU profile attributed 79.7% of samples to `positionAt`. This proved
that repeated position scans made the implementation effectively quadratic for
node-rich pages.

## A/B optimization history

Each accepted change was benchmarked on the complete 10,000-page in-memory
corpus. Changes that did not improve the distribution were reverted.

| Experiment | Result | Decision |
| --- | ---: | --- |
| Build line starts once instead of rescanning from byte zero | About 74% faster | Accept |
| Preallocate node and attribute arenas from source size | Median about 180 ms to 165 ms; 24.5% fewer allocated bytes | Accept |
| Combine the ASCII decision with source indexing | Median about 165 ms to 155 ms | Accept, later replaced by lazy positions |
| Allocation-free JSON scanner | About 12% faster | Accept |
| Closed-schema component switches and source-range duplicate checks | Median about 137 ms to 121 ms; allocations 176,000 to 47,000 | Accept |
| Build the line index only on a diagnostic; track inline positions forward | Median about 121 ms to 106 ms; about 33,000 allocations | Accept |
| 64-bit ASCII scan | About 5% faster | Accept |
| Jump directly to inline delimiter bytes | About 7% faster | Accept |
| Vectorized line search with bounded CR search | Several percent faster and restores a strict linear scan | Accept |
| Size attribute arena from source length | Reached three allocations per valid page | Accept |
| Size node arena from source length | About 3.5% faster and 24.8% less memory | Accept |
| Cache the current line after vectorizing line search | About 2% faster | Accept |
| Static inline delimiter lookup instead of `bytes.IndexAny` | About 4.3% faster | Accept |
| `binary.LittleEndian.Uint64` ASCII words | About 4.1% faster | Accept |
| Store diagnostics only in the document arena | Node size 64 to 56 bytes; about 9% less node memory | Accept |
| Four-record inline reference buffer | Recovered nearly all reference-validation cost without weakening overflow handling | Accept |
| Detect LF-only documents once and skip per-line CR searches | About 4.3% faster | Accept |
| Unroll the fixed inline-delimiter lookup eight bytes at a time | About 2.4% faster | Accept |
| Cache a line with the earlier scalar scanner | About 1.2% slower | Reject |
| Replace line search with `bytes.IndexAny` | About 4.8% slower | Reject |
| Add 16-bit or 32-bit attribute hashes | Median 79.6 ms versus 76.9 ms for exact ranges | Reject |
| Static JSON special-byte lookup | Neutral to slightly slower | Reject |
| Combine ASCII and CR detection in one Go word loop | About 5.5% slower than the standard library byte search | Reject |
| Carry the CR fast-path flag through inline position trackers | About 2% slower | Reject |
| Add a 64-bit duplicate-attribute collision filter | About 1% slower | Reject |
| Use zero instead of `0xffffffff` for tree-link sentinels | About 1.8% slower | Reject |
| Unroll JSON string scanning | Neutral to about 0.6% slower | Reject |
| Use assembly quote and escape searches for JSON strings | Neutral to about 0.4% slower | Reject |

The final profile is no longer dominated by an avoidable algorithm. Its largest
parser-owned flat costs are arena node construction at 9.01%, inline parsing at
5.92%, JSON string validation at 5.15%, ASCII classification at 3.22%, and
delimiter lookup at 2.70%. Each of these areas has either a retained A/B win or
a rejected faster-looking alternative recorded above. Allocation space is the
returned document tree: 98.42% of measured allocation space is in
`ParseWithOptions`. Further tested reductions either moved work elsewhere or
made the complete corpus slower.

## Final comparison

Run this comparison with:

```sh
go test ./internal/mpd -run '^$' \
  -bench '^(BenchmarkParseCorpus10000|BenchmarkGoldmarkCorpus10000)$' \
  -benchtime=10x -count=10 -benchmem
```

Ten final samples produced these distributions:

| Parser | Minimum | Median | Maximum | Median bytes | Median allocations |
| --- | ---: | ---: | ---: | ---: | ---: |
| Native M-Press parser | 58.62 ms | 64.47 ms | 67.02 ms | 47,007,648 | 30,000 |
| Goldmark | 288.55 ms | 324.66 ms | 355.09 ms | 153,346,214 | 1,154,132 |

On this corpus, the native parser is 5.04 times faster than Goldmark. It uses
30.65% of Goldmark's allocated bytes and 2.60% of its allocation count. Each
valid page requires exactly three allocations: the document, node arena,
and attribute arena.

The final native median is about 90.7% lower than the initial 693 ms median.
This comparison is deliberately conservative because Goldmark parses a less
structured language and does not extract the complete source model.

With `GOMAXPROCS=1`, eight samples ranged from 38.54 to 40.87 ms with a median
of 38.92 ms. The parser is single-threaded and has no one-CPU regression. The
lower time comes from reduced concurrent garbage-collector overhead during this
allocation-heavy benchmark.

## Correctness gates

The optimized implementation passed all of these gates:

- every checked-in syntax fixture parses without an error diagnostic;
- the production HTML golden suite remains byte-for-byte unchanged;
- all 10,000 generated on-disk pages parse and all 256 assets verify;
- LF, CRLF, CR, UTF-8, metadata, limits, component nesting, code opacity,
  tables, lists, references, footnotes, and recovery have focused tests;
- escaped JSON reference identifiers and reference counts beyond the inline
  buffer have regression tests;
- unmatched delimiters retain a fully connected syntax tree;
- race tests pass for the parser, corpus, and conformance packages;
- extended parser fuzzing found and pinned two leading-indentation non-progress
  regressions, then the final clean run completed 270,299 executions without a
  panic or invariant failure; and
- the final JSON scanner differential fuzz run completed 371,797 executions against
  `encoding/json.Valid` without a mismatch.

The final profile commands are:

```sh
go test ./internal/mpd -run '^$' -bench '^BenchmarkParseCorpus10000$' \
  -benchtime=100x -cpuprofile=/tmp/mpd.cpu.pprof \
  -memprofile=/tmp/mpd.heap.pprof -benchmem
go tool pprof -top /tmp/mpd.cpu.pprof
go tool pprof -top -alloc_space /tmp/mpd.heap.pprof
```
