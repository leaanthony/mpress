# Third-party notices

Release archives contain `third_party_licenses/INDEX.txt` and every upstream
license, copying, and notice file found for each Go module linked into the
command binary. The bundle is generated from the release dependency graph, so
it cannot silently fall behind `go.mod`. The directory also contains the Astro
Starlight icon license described below.

## D2 diagrams

Fenced D2 diagrams are compiled and rendered using the unmodified
[D2 v0.8.1 source](https://github.com/d2lang/d2/tree/v0.8.1), licensed under
Mozilla Public License 2.0. Release archives include the D2 license and the
licenses of its linked dependencies. The linked source is available from the
upstream repository and the Go module proxy.

Copyright 2022 Terrastruct Inc.

## OpenAI Go SDK

The translation provider client uses OpenAI Go SDK v3. The SDK is licensed
under Apache License 2.0. A complete copy is preserved at
`internal/translate/vendor/openai-go-LICENSE`.

Copyright 2026 OpenAI

## Astro Starlight icons

Selected social and platform icon paths are derived from Astro Starlight.
The complete upstream icon source bundles and license are preserved in
`internal/icons/vendor/`.

MIT License

Copyright (c) 2023 [Astro contributors](https://github.com/withastro/starlight/graphs/contributors)

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Go Brotli

Compression regression tests use
[`andybalholm/brotli`](https://github.com/andybalholm/brotli), version 1.2.2.
It is distributed under the MIT License and is not linked into the M-Press
command binary.

Copyright (c) 2014 Google, Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
